package main

import (
	"flag"
	"fmt"
	"github.com/gabesullice/go-1brc/pkg/report"
	"golang.org/x/exp/mmap"
	"io"
	"os"
	"runtime"
)

const (
	ErrDefault = 1 + iota
	ErrMissingArg
	ErrFileOpen
	ErrStdErr
)

var useMMap bool

func init() {
	flag.BoolVar(&useMMap, "mmap", false, "use mmap")
}

func main() {
	flag.Parse()
	var reader io.ReaderAt
	var size int
	if useMMap {
		if len(os.Args) < 3 {
			exitOnErr(fmt.Errorf("missing file argument"), ErrMissingArg)
		}
		mm, err := mmap.Open(os.Args[2])
		exitOnErr(err, ErrFileOpen)
		reader = mm
		size = mm.Len()
	} else {
		if len(os.Args) < 2 {
			exitOnErr(fmt.Errorf("missing file argument"), ErrMissingArg)
		}
		f, err := os.Open(os.Args[1])
		exitOnErr(err, ErrFileOpen)
		stat, err := f.Stat()
		exitOnErr(err, ErrFileOpen)
		reader = f
		size = int(stat.Size())
	}
	concurrency := runtime.NumCPU() * 2
	exitOnErr(report.Generate(os.Stdout, reader, size, concurrency), ErrDefault)
}

func exitOnErr(err error, code int) {
	if err != nil {
		if _, printErr := fmt.Fprintln(os.Stderr, err); printErr != nil {
			os.Exit(ErrStdErr)
		}
		os.Exit(code)
	}
}
