#include "textflag.h"

// func axpyKernel(dst, src []float32, k float32)
//
// dst[i] += k * src[i] for i < len(dst), with NEON: 16 floats per round
// (four 128-bit registers of 4 floats), then 4 at a time, then one by one.
// Every multiply-add is fused (one rounding), like Go's own dst[i] += k*v on
// arm64, so the results are the same bit for bit.
//
// Go's assembler writes operands source first: "VFMLA V30.S4, V4.S4, V0.S4"
// is V0 += V4 * V30 (vfmaq_f32 in C). Registers: R0 dst, R1 src, R2 count,
// V30 = k in all 4 lanes, F31 = k.
TEXT ·axpyKernel(SB), NOSPLIT, $0-52
	MOVD	dst_base+0(FP), R0         // R0 = &dst[0]
	MOVD	dst_len+8(FP), R2          // R2 = len(dst), floats left to do
	MOVD	src_base+24(FP), R1        // R1 = &src[0]
	FMOVS	k+48(FP), F31              // F31 = k (the low lane of V31)
	VDUP	V31.S[0], V30.S4           // V30 = {k, k, k, k}      vdupq_n_f32(k)

loop16:
	CMP	$16, R2
	BLT	loop4                      // fewer than 16 left
	VLD1	(R0), [V0.S4, V1.S4, V2.S4, V3.S4]      // 16 floats of dst   vld1q_f32 x4
	VLD1.P	64(R1), [V4.S4, V5.S4, V6.S4, V7.S4]    // 16 floats of src, R1 += 64 bytes
	VFMLA	V30.S4, V4.S4, V0.S4       // V0 += src[0:4] * k      vfmaq_f32(V0, V4, V30)
	VFMLA	V30.S4, V5.S4, V1.S4       // V1 += src[4:8] * k
	VFMLA	V30.S4, V6.S4, V2.S4       // V2 += src[8:12] * k
	VFMLA	V30.S4, V7.S4, V3.S4       // V3 += src[12:16] * k
	VST1.P	[V0.S4, V1.S4, V2.S4, V3.S4], 64(R0)    // store into dst, R0 += 64 bytes
	SUB	$16, R2
	B	loop16

loop4:
	CMP	$4, R2
	BLT	tail                       // fewer than 4 left
	VLD1	(R0), [V0.S4]              // 4 floats of dst
	VLD1.P	16(R1), [V4.S4]            // 4 floats of src, R1 += 16 bytes
	VFMLA	V30.S4, V4.S4, V0.S4       // V0 += src * k
	VST1.P	[V0.S4], 16(R0)            // store, R0 += 16 bytes
	SUB	$4, R2
	B	loop4

tail:
	CBZ	R2, done                   // nothing left
	FMOVS	(R0), F0                   // F0 = dst[i]
	FMOVS.P	4(R1), F4                  // F4 = src[i], R1 += 4 bytes
	FMADDS	F31, F0, F4, F0            // F0 = F0 + F4 * F31      fmaf(src, k, dst)
	FMOVS.P	F0, 4(R0)                  // dst[i] = F0, R0 += 4 bytes
	SUB	$1, R2
	B	tail

done:
	RET
