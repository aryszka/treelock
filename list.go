package treelock

type item struct {
	operation  *operation
	prev, next *item
}

type listRange struct {
	first, last *item
}

func (l listRange) empty() bool {
	return l.first == nil
}

func rangeOver(l listRange, f func(*operation)) {
	if l.first == nil {
		return
	}

	i := l.first
	for {
		f(i.operation)
		if i == l.last {
			return
		}

		i = i.next
	}
}

func insertTo(l listRange, i *item) listRange {
	if l.empty() {
		l.first, l.last = i, i
		return l
	}

	if l.first == i.next {
		l.first = i
		return l
	}

	if l.last == i.prev {
		l.last = i
		return l
	}

	if i.prev != nil {
		return l
	}

	if l.last.next != nil {
		i.next = l.last.next
		l.last.next.prev = i
	}

	l.last.next = i
	i.prev = l.last
	l.last = i
	return l
}

func removeFrom(l listRange, i *item) listRange {
	if i.prev != nil {
		i.prev.next = i.next
	}

	if i.next != nil {
		i.next.prev = i.prev
	}

	if l.first == i && l.last == i {
		l.first, l.last = nil, nil
	} else if l.first == i {
		l.first = i.next
	} else if l.last == i {
		l.last = i.prev
	}

	return l
}

func connect(left, right listRange) {
	if left.empty() || right.empty() {
		return
	}

	if left.last.next == right.first {
		return
	}

	if left.last.next != nil {
		left.last.next.prev = right.last
		right.last.next = left.last.next
	}

	if right.first.prev != nil {
		right.first.prev.next = left.first
		left.first.prev = right.first.prev
	}

	left.last.next = right.first
	right.first.prev = left.last
}
