package report

import "slices"

type tree struct {
	root *node
}

func newTree() (t *tree) {
	return new(tree)
}

func (t *tree) add(r *reading) {
	if t.root == nil {
		t.root = newNode(r)
	} else {
		t.root.add(r)
	}
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

func (t *tree) flatten() []*record {
	if t.root == nil {
		return []*record{}
	}
	records := t.root.flatten()
	slices.SortFunc(records, compareRecords)
	return records
}
