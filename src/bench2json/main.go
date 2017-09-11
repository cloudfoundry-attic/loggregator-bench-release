// bench2json: a tool that converts go bench output to json.
//
package main

import (
	"bench2json/internal/scanner"
	"encoding/json"
	"log"
	"os"

	"golang.org/x/tools/benchmark/parse"
)

func main() {
	var bs []*parse.Benchmark
	s := scanner.New(os.Stdin)

	for s.Scan() {
		bs = append(bs, s.Benchmark())
	}

	if err := s.Err(); err != nil {
		log.Fatal(err)
	}

	data, err := json.Marshal(bs)
	if err != nil {
		log.Fatal(err)
	}

	_, _ = os.Stdout.Write(data)
}
