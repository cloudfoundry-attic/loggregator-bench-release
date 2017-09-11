package scanner_test

import (
	"bench2json/internal/scanner"
	"strings"
	"testing"

	"golang.org/x/tools/benchmark/parse"
)

func TestScanner(t *testing.T) {
	t.Parallel()

	r := strings.NewReader(`
BenchmarkThroughput-20          2000000000               1.00 ns/op            3 B/op          0 allocs/op
BenchmarkLatency-20             2000000000               2.00 ns/op            4 B/op          0 allocs/op
PASS
ok      doppler 0.007s
`)
	s := scanner.New(r)

	var results []*parse.Benchmark
	for s.Scan() {
		results = append(results, s.Benchmark())
	}

	if s.Err() != nil {
		t.Fatal(s.Err())
	}

	if len(results) != 2 {
		t.Fatalf("Expected %d to equal %d", len(results), 2)
	}
}

func TestScannerBadInput(t *testing.T) {
	t.Parallel()

	r := strings.NewReader(`
BenchmarkGarbage
PASS
ok      doppler 0.007s
`)
	s := scanner.New(r)

	for s.Scan() {
		// NOP
	}
	if s.Err() == nil {
		t.Fatal("Expected an error to have occurred")
	}
}
