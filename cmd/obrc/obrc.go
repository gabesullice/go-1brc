package main

import (
	"fmt"
	"github.com/gabesullice/obrc-go/pkg/report"
	"os"
)

const (
	ErrDefault = 1 + iota
	ErrMissingArg
	ErrFileOpen
	ErrStdErr
)

func main() {
	if len(os.Args) < 2 {
		exitOnErr(fmt.Errorf("missing file argument"), ErrMissingArg)
	}
	f, err := os.Open(os.Args[1])
	exitOnErr(err, ErrFileOpen)
	exitOnErr(report.Generate(f, os.Stdout), ErrDefault)
}

func exitOnErr(err error, code int) {
	if err != nil {
		if _, printErr := fmt.Fprintln(os.Stderr, err); printErr != nil {
			os.Exit(ErrStdErr)
		}
		os.Exit(code)
	}
}
