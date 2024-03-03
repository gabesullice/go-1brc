package report

import (
	"bytes"
	"fmt"
	"io"
	"slices"
)

type reading struct {
	station     []byte
	stationHash uint64
	temperature int64
}

const noNewline = -1

const lenMinReading = len("A;0.0\n")

const (
	fnvOffsetBasis uint64 = 14695981039346656037
	fnvPrime       uint64 = 1099511628211
)

type record struct {
	name          []byte
	min, max, sum int64
	count         uint64
}

func compareRecords(a, b *record) int {
	return bytes.Compare(a.name, b.name)
}

type tree struct {
	root *node
}

func (t *tree) flatten() []*record {
	if t.root == nil {
		return []*record{}
	}
	records := t.root.flatten()
	slices.SortFunc(records, compareRecords)
	return records
}

type node struct {
	hash        uint64
	left, right *node
	record      *record
}

func newNode(r *reading) *node {
	n := &node{
		hash: r.stationHash,
		record: &record{
			name:  slices.Clone(r.station),
			count: 1,
		},
	}
	n.record.min = r.temperature
	n.record.max = r.temperature
	n.record.sum = r.temperature
	return n
}

func (n *node) flatten() (records []*record) {
	if n.left != nil {
		records = append(records, n.left.flatten()...)
	}
	if n.right != nil {
		records = append(records, n.right.flatten()...)
	}
	return append(records, n.record)
}

func (n *node) add(r *reading) {
	if r.stationHash < n.hash {
		if n.left == nil {
			n.left = newNode(r)
		} else {
			n.left.add(r)
		}
		return
	}
	if r.stationHash > n.hash {
		if n.right == nil {
			n.right = newNode(r)
		} else {
			n.right.add(r)
		}
		return
	}
	if r.temperature < n.record.min {
		n.record.min = r.temperature
	} else if r.temperature > n.record.max {
		n.record.max = r.temperature
	}
	n.record.sum += r.temperature
	n.record.count++
}

const maxReadLength = 2 << 13

func parseFile(f io.ReaderAt, size int64, buf []byte, readings *tree) {
	parseFileLeftRight(f, 0, int(size), buf, readings)
}

func parseFileLeftRight(f io.ReaderAt, left, right int, buf []byte, readings *tree) int {
	size := right - left
	if size <= cap(buf) {
		buf = buf[:size]
		if _, err := f.ReadAt(buf, int64(left)); err != nil {
			panic(err)
		}
		tnl := left + parseBytes(buf, readings)
		return tnl
	}
	half := size / 2
	var splitAt int
	if half > cap(buf) {
		splitAt = left + half - (half % cap(buf))
	} else {
		splitAt = left + half
	}
	leftTNL := parseFileLeftRight(f, left, splitAt, buf, readings)
	return parseFileLeftRight(f, leftTNL+1, right, buf, readings)
}

func parseBytes(d []byte, readings *tree) (terminalNL int) {
	if len(d) < lenMinReading {
		panic(fmt.Sprintf("too few bytes: \"%s\"", d))
	}
	i := len(d) - 1
	// Ignore anything after the terminal newline in the byte slice.
	terminalNL = noNewline
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
	// TODO: test if instantiating this as a pointer improves performance.
	parsed := reading{}
	var saveReading func()
	saveReading = func() {
		if readings.root == nil {
			readings.root = newNode(&parsed)
		} else {
			readings.root.add(&parsed)
		}
		saveReading = func() {
			readings.root.add(&parsed)
		}
	}
nextReading:
	// Tenths
	temp := d[i] &^ '0'
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
			saveReading()
			i--
			goto nextReading
		}
		parsed.stationHash *= fnvPrime
		parsed.stationHash ^= uint64(d[i])
	}
	parsed.station = d[:semicolonIndex]
	saveReading()
	return terminalNL
}
