// Command samples turns the handwriting the crossword logged (samples.jsonl:
// letters it misread and the user wrote again) into training records like
// synth's: 28x28 bytes (ink 255) followed by the label (0 = A). Both writings
// of each letter are kept.
//
//	scp remarkable:.config/crossword/samples.jsonl build/hwr/
//	go run ./games/crossword/hwrtrain/samples -o build/hwr/real.bin build/hwr/samples.jsonl
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"appload-games/games/crossword/backend/hwr"
)

type writing struct {
	Strokes [][]float64 `json:"strokes"`
	Guesses []hwr.Guess `json:"guesses"`
}

func main() {
	out := flag.String("o", "real.bin", "output file")
	flag.Parse()

	f, err := os.Create(*out)
	if err != nil {
		log.Fatal(err)
	}
	w := bufio.NewWriter(f)
	rec := make([]byte, hwr.Size*hwr.Size+1)
	count := map[string]int{}
	for _, path := range flag.Args() {
		in, err := os.Open(path)
		if err != nil {
			log.Fatal(err)
		}
		sc := bufio.NewScanner(in)
		sc.Buffer(make([]byte, 1<<20), 1<<20)
		for sc.Scan() {
			var s struct {
				Letter  string  `json:"letter"`
				Misread writing `json:"misread"`
				Fixed   writing `json:"fixed"`
			}
			if err := json.Unmarshal(sc.Bytes(), &s); err != nil || len(s.Letter) != 1 || s.Letter[0] < 'A' || s.Letter[0] > 'Z' {
				continue
			}
			for _, wr := range []writing{s.Misread, s.Fixed} {
				var strokes []hwr.Stroke
				for _, flat := range wr.Strokes {
					var st hwr.Stroke
					for i := 0; i+1 < len(flat); i += 2 {
						st = append(st, hwr.Point{X: flat[i], Y: flat[i+1]})
					}
					strokes = append(strokes, st)
				}
				img, ok := hwr.Image(strokes)
				if !ok {
					continue
				}
				for k, v := range img {
					rec[k] = byte(v*255 + 0.5)
				}
				rec[len(rec)-1] = s.Letter[0] - 'A'
				w.Write(rec)
			}
			count[s.Letter]++
		}
		in.Close()
	}
	if err := w.Flush(); err != nil {
		log.Fatal(err)
	}
	f.Close()
	fmt.Println("samples per letter:", count)
}
