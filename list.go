package treelock

type element struct {
	item       *item
	prev, next *element
}

type list struct {
	first, last *element
}

func (l list) empty() bool {
	return l.first == nil
}

func (l list) rangeOver(f func(*item)) {
	if l.first == nil {
		return
	}

	e := l.first
	for {
		f(e.item)
		if e == l.last {
			return
		}

		e = e.next
	}
}

func (l list) insert(e *element) list {
	if l.empty() {
		l.first, l.last = e, e
		return l
	}

	if l.first == e.next {
		l.first = e
		return l
	}

	if l.last == e.prev {
		l.last = e
		return l
	}

	if e.prev != nil {
		return l
	}

	if l.last.next != nil {
		e.next = l.last.next
		l.last.next.prev = e
	}

	l.last.next = e
	e.prev = l.last
	l.last = e
	return l
}

func (l list) remove(e *element) list {
	if e.prev != nil {
		e.prev.next = e.next
	}

	if e.next != nil {
		e.next.prev = e.prev
	}

	if l.first == e && l.last == e {
		l.first, l.last = nil, nil
	} else if l.first == e {
		l.first = e.next
	} else if l.last == e {
		l.last = e.prev
	}

	return l
}

func connect(left, right list) {
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

	if right.first.prev == nil {
		left.last.next = right.first
		right.first.prev = left.last
		return
	}

	right.first.prev.next = left.first
	left.first.prev = right.first.prev
	left.last.next = right.first
	right.first.prev = left.last
	return
}
