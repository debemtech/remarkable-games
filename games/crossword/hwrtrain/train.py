#!/usr/bin/env python3
"""Trains the letter recognizer and writes games/crossword/backend/hwr/model.bin.

Needs PyTorch and numpy, the EMNIST letters split (gzip IDX files) and the
synthetic pen letters from ./synth:

    go run ./games/crossword/hwrtrain/synth -n 78000 -o build/hwr/synth.bin
    python3 games/crossword/hwrtrain/train.py --emnist path/to/emnist --synth build/hwr/synth.bin

The model is small on purpose: two convolutions and two dense layers, about
210k weights, stored as int8 (about 210 KB).
"""
import argparse
import gzip
import os
import struct

import numpy as np
import torch
import torch.nn as nn
import torch.nn.functional as F

HERE = os.path.dirname(os.path.abspath(__file__))
OUT = os.path.join(HERE, "..", "backend", "hwr", "model.bin")


def load_idx(path):
    data = gzip.open(path).read()
    if data[2] == 8 and data[3] == 3:  # images
        n, h, w = struct.unpack(">III", data[4:16])
        return np.frombuffer(data, np.uint8, offset=16).reshape(n, h, w)
    n = struct.unpack(">I", data[4:8])[0]
    return np.frombuffer(data, np.uint8, offset=8)


def emnist(folder, split):
    imgs = load_idx(os.path.join(folder, f"emnist-letters-{split}-images-idx3-ubyte.gz"))
    labels = load_idx(os.path.join(folder, f"emnist-letters-{split}-labels-idx1-ubyte.gz"))
    # EMNIST images are stored transposed; labels are 1 (A) to 26 (Z).
    return imgs.transpose(0, 2, 1).copy(), labels.astype(np.int64) - 1


def synth(path):
    d = np.fromfile(path, np.uint8).reshape(-1, 785)
    return d[:, :784].reshape(-1, 28, 28).copy(), d[:, 784].astype(np.int64)


class Net(nn.Module):
    def __init__(self):
        super().__init__()
        self.c1 = nn.Conv2d(1, 16, 3, padding=1)
        self.c2 = nn.Conv2d(16, 32, 3, padding=1)
        self.f1 = nn.Linear(32 * 7 * 7, 128)
        self.f2 = nn.Linear(128, 26)

    def forward(self, x):
        x = F.max_pool2d(F.relu(self.c1(x)), 2)
        x = F.max_pool2d(F.relu(self.c2(x)), 2)
        x = F.relu(self.f1(x.flatten(1)))
        return self.f2(x)


def augment(x):
    """Random rotation, scale, shear and shift, as people write letters."""
    n = x.shape[0]
    angle = (torch.rand(n, device=x.device) * 2 - 1) * 0.2
    scale = 0.85 + torch.rand(n, device=x.device) * 0.3
    shear = (torch.rand(n, device=x.device) * 2 - 1) * 0.2
    tx = (torch.rand(n, device=x.device) * 2 - 1) * 0.1
    ty = (torch.rand(n, device=x.device) * 2 - 1) * 0.1
    cos, sin = torch.cos(angle) / scale, torch.sin(angle) / scale
    theta = torch.stack([torch.stack([cos, -sin + shear, tx], 1), torch.stack([sin, cos, ty], 1)], 1)
    grid = F.affine_grid(theta, x.shape, align_corners=False)
    return F.grid_sample(x, grid, align_corners=False)


def accuracy(model, x, y, device):
    model.eval()
    correct = 0
    with torch.no_grad():
        for i in range(0, len(x), 4096):
            xb = torch.from_numpy(x[i:i + 4096]).float().div(255).unsqueeze(1).to(device)
            correct += (model(xb).argmax(1).cpu().numpy() == y[i:i + 4096]).sum()
    model.train()
    return correct / len(x)


def quantize(t):
    a = t.detach().cpu().numpy().astype(np.float32).ravel()
    scale = max(np.abs(a).max() / 127, 1e-8)
    return np.clip(np.round(a / scale), -127, 127).astype(np.int8), np.float32(scale)


def export(model, path):
    """model.bin: "HWR1", then per tensor: u32 length, f32 scale, int8 values.
    Order: c1.w c1.b c2.w c2.b f1.w f1.b f2.w f2.b (PyTorch layouts)."""
    with open(path, "wb") as f:
        f.write(b"HWR1")
        for layer in (model.c1, model.c2, model.f1, model.f2):
            for t in (layer.weight, layer.bias):
                q, scale = quantize(t)
                f.write(struct.pack("<If", len(q), scale))
                f.write(q.tobytes())


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--emnist", required=True)
    ap.add_argument("--synth", required=True)
    ap.add_argument("--epochs", type=int, default=12)
    ap.add_argument("--out", default=OUT)
    args = ap.parse_args()
    torch.manual_seed(1)
    device = "cuda" if torch.cuda.is_available() else "cpu"

    ex, ey = emnist(args.emnist, "train")
    tx, ty = emnist(args.emnist, "test")
    sx, sy = synth(args.synth)
    split = len(sx) * 9 // 10
    x = np.concatenate([ex, sx[:split]])
    y = np.concatenate([ey, sy[:split]])
    print(f"training on {len(x)} letters ({len(ex)} EMNIST, {split} synthetic) on {device}")

    model = Net().to(device)
    opt = torch.optim.Adam(model.parameters(), lr=2e-3)
    sched = torch.optim.lr_scheduler.OneCycleLR(opt, 3e-3, total_steps=args.epochs * (len(x) // 256 + 1))
    X = torch.from_numpy(x).float().div(255).unsqueeze(1)
    Y = torch.from_numpy(y)
    for epoch in range(args.epochs):
        perm = torch.randperm(len(X))
        for i in range(0, len(X), 256):
            idx = perm[i:i + 256]
            xb, yb = augment(X[idx].to(device)), Y[idx].to(device)
            loss = F.cross_entropy(model(xb), yb, label_smoothing=0.05)
            opt.zero_grad()
            loss.backward()
            opt.step()
            sched.step()
        print(f"epoch {epoch + 1}: loss {loss.item():.3f}, EMNIST test {accuracy(model, tx, ty, device):.4f}, "
              f"synthetic test {accuracy(model, sx[split:], sy[split:], device):.4f}")

    export(model, args.out)
    print("wrote", os.path.relpath(args.out), os.path.getsize(args.out), "bytes")

    # Reference outputs for the Go test, from the float model and from the int8 model
    q = Net()
    q.load_state_dict({k: torch.from_numpy(quantize(v)[0].astype(np.float32) * quantize(v)[1]).reshape(v.shape)
                       for k, v in model.state_dict().items()})
    sample = torch.from_numpy(sx[split:split + 8]).float().div(255).unsqueeze(1)
    with torch.no_grad():
        logits = q(sample).numpy()
    ref = os.path.join(os.path.dirname(args.out), "testdata")
    os.makedirs(ref, exist_ok=True)
    with open(os.path.join(ref, "reference.bin"), "wb") as f:
        f.write(struct.pack("<I", len(sample)))
        f.write(sx[split:split + 8].tobytes())
        f.write(logits.astype("<f4").tobytes())
    print("int8 model: EMNIST test", accuracy(q.to(device), tx, ty, device))


if __name__ == "__main__":
    main()
