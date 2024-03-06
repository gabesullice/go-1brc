package report

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"slices"
	"sync"
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

func (r *record) String() string {
	var mean float64
	minimum, maximum := float64(r.min), float64(r.max)
	sum, count := int(r.sum), int(r.count)
	if sum%count != 0 {
		mean = float64(sum/count + 1)
	} else {
		mean = float64(sum / count)
	}
	return fmt.Sprintf("%s=%.1f/%.1f/%.1f", r.name, minimum/10, mean/10, maximum/10)
}

func (r *record) add(other *record) {
	if !bytes.Equal(other.name, r.name) {
		panic("records do not match")
	}
	if other.min < r.min {
		r.min = other.min
	}
	if other.max > r.max {
		r.max = other.max
	}
	r.sum += other.sum
	r.count += other.count
}

func compareRecords(a, b *record) int {
	return bytes.Compare(a.name, b.name)
}

type tree struct {
	root *node
}

func (t *tree) merge(other *tree) *tree {
	if other.root == nil {
		return t
	}
	if t.root == nil {
		return other
	}
	t.root = merge(t.root, other.root)
	return t
}

func merge(a, b *node) *node {
	if b.left != nil {
		a = merge(a, b.left)
		b.left = nil
	}
	if b.right != nil {
		a = merge(a, b.right)
		b.right = nil
	}
	a.insert(b)
	return a
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

func (n *node) insert(other *node) {
	if other.left != nil || other.right != nil {
		panic("other node must not have children")
	}
	if other.hash < n.hash {
		if n.left == nil {
			n.left = other
		} else {
			n.left.insert(other)
		}
		return
	}
	if other.hash > n.hash {
		if n.right == nil {
			n.right = other
		} else {
			n.right.insert(other)
		}
		return
	}
	n.record.add(other.record)
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

const concurrency = 2<<2 - 1

func Generate(f *os.File, out io.Writer) error {
	readings := parseFile(f)
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
	for _ = range concurrency {
		wg.Add(1)
		clip := bytesAfterLastByte(f, offset+chunkSize, '\n')
		t := new(tree)
		go func(offset, chunkSize, clip int, readings *tree) {
			buf := make([]byte, 0, maxReadLength)
			parseFileLeftRight(f, offset, offset+chunkSize-clip, buf, readings)
			wg.Done()
		}(offset, chunkSize, clip, t)
		trees = append(trees, t)
		offset += chunkSize - clip
	}
	readings := new(tree)
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
	buf := make([]byte, end)
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
