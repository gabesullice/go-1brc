package report

type node struct {
	hash        uint32
	left, right *node
	record      *record
}

func newNode(r *reading) *node {
	return &node{
		hash:   r.stationHash,
		record: newRecord(r),
	}
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
