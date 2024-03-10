package report

import (
	"slices"
)

type hashMap struct {
	size  uint32
	array [hashMapSize]*tree
}

func newHashMap(size uint32) *hashMap {
	var arr [hashMapSize]*tree
	for i := range arr {
		arr[i] = newTree()
	}
	return &hashMap{
		size:  size,
		array: arr,
	}
}

func (m *hashMap) add(r *reading) {
	m.array[r.stationHash%m.size].add(r)
}

func (m *hashMap) flatten() []*record {
	records := make([]*record, 0, m.size)
	for _, rs := range m.array {
		records = append(records, rs.flatten()...)
	}
	slices.SortFunc(records, compareRecords)
	return records
}

func (m *hashMap) merge(other *hashMap) *hashMap {
	merged := newHashMap(m.size)
	for i := range merged.array {
		merged.array[i] = m.array[i].merge(other.array[i])
	}
	return merged
}
