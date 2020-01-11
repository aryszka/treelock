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

func (l list) rangeOver(f func(*element)) {
	if l.first == nil {
		return
	}

	e := l.first
	for {
		f(e)
		if e == l.last {
			return
		}

		e = e.next
	}
}

// if the element cannot be inserted, it's a noop
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

	if l.last.next != nil {
		e.next = l.last.next
		l.last.next.prev = e
	}

	l.last.next = e
	e.prev = l.last
	l.last = e
	return l
}

func connect(left, right list) {
	if left.empty() || right.empty() {
		return
	}

	left.last.next, right.first.prev = right.first, left.last
}
