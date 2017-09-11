package scanner

import (
	"bufio"
	"io"
	"strings"

	"golang.org/x/tools/benchmark/parse"
)

// Scanner follows the scanner pattern in bufio but for *parse.Benchmark.
type Scanner struct {
	scanner *bufio.Scanner
	b       *parse.Benchmark
	err     error
}

// New creates a new Scanner.
func New(r io.Reader) *Scanner {
	return &Scanner{
		scanner: bufio.NewScanner(r),
	}
}

// Scan moves the scanner forward a line.
func (s *Scanner) Scan() bool {
	for {
		if s.err != nil || !s.scanner.Scan() {
			return false
		}
		line := s.scanner.Text()
		if strings.HasPrefix(line, "Benchmark") {
			return s.parseBenchmark()
		}
	}
}

func (s *Scanner) parseBenchmark() bool {
	line := s.scanner.Text()
	s.b, s.err = parse.ParseLine(line)
	return s.err == nil
}

// Err returns if an error occured, either with parsing or reading.
func (s *Scanner) Err() error {
	if s.err != nil {
		return s.err
	}
	return s.scanner.Err()
}

// Benchmark should be called when Scan returns true in order to obtain the
// current *parse.Benchmark.
func (s *Scanner) Benchmark() *parse.Benchmark {
	return s.b
}
