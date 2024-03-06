package report

import (
	"io"
)

func Generate(out io.Writer, f io.ReaderAt, size int, concurrency int) error {
	readings := parseFile(f, size, concurrency)
	if _, err := out.Write([]byte("{")); err != nil {
		return err
	}
	records := readings.flatten()
	count := len(records)
	if count >= 1 {
		for i := range count - 1 {
			rec := records[i].String() + ", "
			if _, err := out.Write([]byte(rec)); err != nil {
				return err
			}
		}
		rec := records[count-1].String()
		if _, err := out.Write([]byte(rec)); err != nil {
			return err
		}
	}
	if _, err := out.Write([]byte("}")); err != nil {
		return err
	}
	return nil
}
