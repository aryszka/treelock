package treelock

type item struct {
	typ       lockType
	path      []string
	notify    chan releaseLock
	element   *element
	blockedBy int
	blocking  []*item
}

type node struct {
	items        list
	subtreeItems list
	children     map[string]*node
}

type tree struct {
	root *node
}

func newTree() *tree {
	return &tree{&node{}}
}

func (t *tree) nodePath(path []string) []*node {
	np := []*node{t.root}
	for len(path) > 0 {
		n, ok := np[len(np)-1].children[path[0]]
		if !ok {
			if np[len(np)-1].children == nil {
				np[len(np)-1].children = make(map[string]*node)
			}

			n = &node{}
			np[len(np)-1].children[path[0]] = n
		}

		np = append(np, n)
		path = path[1:]
	}

	return np
}

func (t *tree) addElement(e *element) {
	var (
		np []*node
		n  *node
	)

	np = t.nodePath(e.item.path)
	n, np = np[len(np)-1], np[:len(np)-1]
	if n.items.empty() {
		n.items.first, n.items.last = e, e
		if n.subtreeItems.first != nil {
			if n.subtreeItems.first.prev != nil {
				n.subtreeItems.first.prev.next = e
				e.prev = n.subtreeItems.first.prev
			}

			e.next = n.subtreeItems.first
			n.subtreeItems.first.prev = e
		}
	} else {
		if n.subtreeItems.empty() {
			if n.items.last.next != nil {
				n.items.last.next.prev = e
				e.next = n.items.last.next
			}
		} else {
			n.subtreeItems.first.prev = e
			e.next = n.subtreeItems.first
		}

		n.items.last.next = e
		e.prev = n.items.last
		n.items.last = e
	}

	for {
		if len(np) == 0 {
			break
		}

		n, np = np[len(np)-1], np[:len(np)-1]
		if n.subtreeItems.empty() {
			n.subtreeItems.first, n.subtreeItems.last = e, e
			if n.items.first != nil {
				if n.items.last.next != nil {
					n.items.last.next.prev = e
					e.next = n.items.last.next
				}

				n.items.last.next = e
				e.prev = n.items.last
			}

			continue
		}

		if n.subtreeItems.first == e.next {
			n.subtreeItems.first = e
			continue
		}

		if n.subtreeItems.last == e.prev {
			n.subtreeItems.last = e
			continue
		}

		if e.next == nil && e.prev == nil {
			if n.subtreeItems.last.next != nil {
				n.subtreeItems.last.next.prev = e
				e.next = n.subtreeItems.last.next
			}

			n.subtreeItems.last.next = e
			e.prev = n.subtreeItems.last
			n.subtreeItems.last = e
			continue
		}

		break
	}
}

func (t *tree) removeElement(e *element) {
	var (
		np []*node
		n  *node
		p  []string
	)

	if e.prev != nil {
		e.prev.next = e.next
	}

	if e.next != nil {
		e.next.prev = e.prev
	}

	np = t.nodePath(e.item.path)
	n = np[len(np)-1]
	if n.items.first == e && n.items.last == e {
		n.items.first, n.items.last = nil, nil
	} else if n.items.first == e {
		n.items.first = e.next
	} else if n.items.last == e {
		n.items.last = e.prev
	}

	p = e.item.path
	for {
		if len(np) == 0 {
			break
		}

		n, np = np[len(np)-1], np[:len(np)-1]
		if n.subtreeItems.first == e && n.subtreeItems.last == e {
			n.subtreeItems.first, n.subtreeItems.last = nil, nil
		} else if n.subtreeItems.first == e {
			n.subtreeItems.first = e.next
		} else if n.subtreeItems.last == e {
			n.subtreeItems.last = e.prev
		}

		if len(np) > 0 && n.items.empty() && n.subtreeItems.empty() {
			delete(np[len(np)-1].children, p[len(p)-1])
		}

		if len(p) > 0 {
			p = p[:len(p)-1]
		}
	}
}
