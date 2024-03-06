package report

import (
	"bytes"
	"fmt"
	"slices"
)

type record struct {
	name          []byte
	min, max, sum int64
	count         uint64
}

func newRecord(r *reading) *record {
	return &record{
		name:  slices.Clone(r.station),
		count: 1,
		min:   r.temperature,
		max:   r.temperature,
		sum:   r.temperature,
	}
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
