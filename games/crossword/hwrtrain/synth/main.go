// Command synth writes synthetic pen-stroke letters as training data for the
// recognizer: records of 28x28 bytes (ink 255) followed by the label (0 = A).
//
//	go run ./games/crossword/hwrtrain/synth -n 80000 -o build/hwr/synth.bin
package main

import (
	"bufio"
	"flag"
	"log"
	"math/rand/v2"
	"os"

	"appload-games/games/crossword/backend/hwr"
	"appload-games/games/crossword/hwrtrain/letters"
)

func main() {
	n := flag.Int("n", 80000, "number of samples")
	out := flag.String("o", "synth.bin", "output file")
	seed := flag.Uint64("seed", 1, "random seed")
	flag.Parse()

	f, err := os.Create(*out)
	if err != nil {
		log.Fatal(err)
	}
	w := bufio.NewWriter(f)
	rng := rand.New(rand.NewPCG(*seed, *seed))
	rec := make([]byte, hwr.Size*hwr.Size+1)
	for i := range *n {
		label := byte(i % 26)
		img, _ := hwr.Image(letters.Variant(rng, 'A'+label))
		for k, v := range img {
			rec[k] = byte(v*255 + 0.5)
		}
		rec[len(rec)-1] = label
		w.Write(rec)
	}
	if err := w.Flush(); err != nil {
		log.Fatal(err)
	}
	f.Close()
}
