package report

import (
	"io"
	"os"
	"sync"
)

const maxReadLength = 2 << 13

const concurrency = 2<<2 - 1

const lenMinReading = len("A;0.0\n")

const (
	fnvOffsetBasis uint64 = 14695981039346656037
	fnvPrime       uint64 = 1099511628211
)

func parseFile(f *os.File) *tree {
	stat, err := f.Stat()
	if err != nil {
		panic(err)
	}
	size := stat.Size()
	chunkSize := int(size / concurrency)
	var offset int
	wg := sync.WaitGroup{}
	trees := make([]*tree, 0, concurrency)
	for range concurrency {
		wg.Add(1)
		clip := bytesAfterLastByte(f, offset+chunkSize, '\n')
		t := newTree()
		go func(offset, chunkSize, clip int, readings *tree) {
			buf := make([]byte, 0, maxReadLength)
			parseFileLeftRight(f, offset, offset+chunkSize-clip, buf, readings)
			wg.Done()
		}(offset, chunkSize, clip, t)
		trees = append(trees, t)
		offset += chunkSize - clip
	}
	readings := newTree()
	buf := make([]byte, 0, maxReadLength)
	parseFileLeftRight(f, offset, int(size), buf, readings)
	wg.Wait()
	for _, t := range trees {
		readings.merge(t)
	}
	return readings
}

func bytesAfterLastByte(r io.ReaderAt, end int, b byte) (count int) {
	bufSize := 2 << 8
	if end < bufSize {
		bufSize = end
	}
	buf := make([]byte, bufSize)
	if _, err := r.ReadAt(buf, int64(end-cap(buf))); err != nil {
		panic(err)
	}
	for i := len(buf) - 1; i >= 0; i-- {
		if buf[i] == b {
			return count
		}
		count++
	}
	panic("not found")
}

func parseComplete(f io.ReaderAt, size int, buf []byte, readings *tree) {
	parseFileLeftRight(f, 0, size, buf, readings)
}

func parseFileLeftRight(f io.ReaderAt, left, right int, buf []byte, readings *tree) int {
	size := right - left
	if size <= maxReadLength {
		buf = buf[:size]
		if _, err := f.ReadAt(buf, int64(left)); err != nil {
			panic(err)
		}
		return left + parseBytes(buf, readings)
	}
	half := size / 2
	var splitAt int
	if half > maxReadLength {
		splitAt = left + half - (half % maxReadLength)
	} else {
		splitAt = left + half
	}
	return parseFileLeftRight(f, parseFileLeftRight(f, left, splitAt, buf, readings)+1, right, buf, readings)
}

func parseBytes(d []byte, readings *tree) (terminalNL int) {
	i := len(d) - 1
	// Ignore anything after the terminal newline in the byte slice.
	for ; i > 0; i-- {
		if d[i] == '\n' {
			terminalNL = i
			i--
			break
		}
	}
	if i == 0 {
		return terminalNL
	}
	if lenMinReading-i > 2 {
		return terminalNL
	}
	var semicolonIndex int
	var temp uint8
	parsed := new(reading)
nextReading:
	// Tenths
	temp = d[i] &^ '0'
	i -= 2 // skip the dot
	// Ones
	temp += d[i] &^ '0' * 10
	i--
	// If a semicolon, there aren't any more temperature bytes to parse, skip to parsing the name.
	if d[i] == ';' {
		parsed.temperature = int64(temp)
		goto consumeName
	}
	// Either a minus or a number in the tens place.
	if d[i] != '-' {
		parsed.temperature = int64(d[i]&^'0')*100 + int64(temp)
		i--
	} else {
		parsed.temperature = int64(temp)
	}
	// Must either be a hyphen-minus or semicolon.
	if d[i] == '-' {
		// It's a hyphen-minus, so the temp is negative.
		parsed.temperature *= -1
		i--
	}
consumeName:
	// d[i] must be a semicolon at this point.
	semicolonIndex = i
	i--
	parsed.stationHash = fnvOffsetBasis
	for ; i >= 0; i-- {
		if d[i] == '\n' {
			parsed.station = d[i+1 : semicolonIndex]
			readings.add(parsed)
			i--
			goto nextReading
		}
		parsed.stationHash *= fnvPrime
		parsed.stationHash ^= uint64(d[i])
	}
	parsed.station = d[:semicolonIndex]
	readings.add(parsed)
	return terminalNL
}
