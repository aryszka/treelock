package treelock

import "strings"

type item struct {
	typ        lockType
	path       []string
	notify     chan releaseLock
	blockedBy  []*item
	blocking   []*item
	node       *node
	prev, next *item
}

type itemList struct {
	first, last *item
}

type node struct {
	children   map[string]*node
	nodeItems  []*item
	items      itemList
	childItems itemList
}

type tree struct {
	root *node
}

func (l itemList) rangeOver(f func(*item)) {
	if l.first == nil {
		return
	}

	i := l.first
	for {
		f(i)
		if i == l.last {
			return
		}

		i = i.next
	}
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

func checkComplete(l itemList, opening, closing bool) bool {
	if l.first == nil && l.last == nil {
		return true
	}

	if l.first == nil || l.last == nil {
		println("one nil")
		return false
	}

	check := l.first
	if opening && check.prev != nil {
		println("opening wrong")
		return false
	}

	for {
		if check == l.last {
			break
		}

		if check.next == check || check.next == nil || check.next.prev != check {
			return false
		}

		check = check.next
	}

	if closing && check.next != nil {
		println("closing wrong")
		return false
	}

	return true
}

func checkConnected(l1, l2 itemList) bool {
	if l1.first == nil || l2.first == nil {
		return true
	}

	if l1.last.next != l2.first || l2.first.prev != l1.last {
		return false
	}

	return true
}

func checkLevel(n *node, path []string) {
	println("checking")
	p := "/" + strings.Join(path, "/")
	if !checkComplete(n.items, false, false) {
		panic("add: items not complete: " + p)
	}

	if !checkComplete(n.childItems, false, false) {
		panic("add: child items not complete: " + p)
	}

	if !checkConnected(n.items, n.childItems) {
		panic("add: items and child items not connected: " + p)
	}
}

func find(l itemList, i *item) {
	if l.first == nil {
		panic("item not found, list nil")
	}

	ii := l.first
	for {
		if ii == i {
			return
		}

		if ii == l.last {
			panic("item not found")
		}

		ii = ii.next
	}
}

func (t *tree) addItem(i *item) {
	var (
		np []*node
		n  *node
	)

	np = t.nodePath(i.path)
	// println("node path length", len(np))
	n, np = np[len(np)-1], np[:len(np)-1]
	if n.items.first == nil {
		if n.childItems.first == nil {
			// println("no items, no child items")
			n.items.first, n.items.last = i, i
		} else {
			if n.childItems.first.prev == nil {
				// println("no items, has child items, sibling has no items")
				n.items.first, n.items.last = i, i
				n.childItems.first.prev = i
				i.next = n.childItems.first
			} else {
				// println("no items, has child items, sibling has items")
				n.items.first, n.items.last = i, i
				n.childItems.first.prev.next = i
				i.prev = n.childItems.first.prev
				n.childItems.first.prev = i
				i.next = n.childItems.first
			}
		}
	} else {
		if n.childItems.first == nil {
			if n.items.last.next == nil {
				// println("has items, no child items, sibling has no items")
				n.items.last.next = i
				i.prev = n.items.last
				n.items.last = i
			} else {
				// println("has items, no child items, sibling has items")
				n.items.last.next.prev = i
				i.next = n.items.last.next
				n.items.last.next = i
				i.prev = n.items.last
				n.items.last = i
			}
		} else {
			// println("has items, has child items")
			n.items.last.next = i
			i.prev = n.items.last
			n.items.last = i
			n.childItems.first.prev = i
			i.next = n.childItems.first
		}
	}

	// p := i.path
	for {
		if len(np) == 0 {
			break
		}

		n, np = np[len(np)-1], np[:len(np)-1]
		if n.childItems.first == nil {
			n.childItems.first, n.childItems.last = i, i
			if n.items.first != nil {
				if i.prev != nil {
					panic("inconsistency in children")
				}

				if n.items.last.next == nil {
					n.items.last.next = i
					i.prev = n.items.last
				} else {
					n.items.last.next.prev = i
					i.next = n.items.last.next
					n.items.last.next = i
					i.prev = n.items.last
				}
			}

			/*
				println("checking single child")
				checkLevel(n, p)
				if len(p) > 0 {
					p = p[:len(p)-1]
				}
			*/

			continue
		}

		if n.childItems.first == i.next {
			n.childItems.first = i

			/*
				println("checking first")
				checkLevel(n, p)
				if len(p) > 0 {
					p = p[:len(p)-1]
				}
			*/

			continue
		}

		if n.childItems.last == i.prev {
			n.childItems.last = i

			/*
				println("checking last")
				checkLevel(n, p)
				if len(p) > 0 {
					p = p[:len(p)-1]
				}
			*/

			continue
		}

		if i.next == nil && i.prev == nil {
			if n.childItems.last.next == nil {
				n.childItems.last.next = i
				i.prev = n.childItems.last
				n.childItems.last = i

				/*
					println("checking new branch, append")
					checkLevel(n, p)
					if len(p) > 0 {
						p = p[:len(p)-1]
					}
				*/

				continue
			} else {
				n.childItems.last.next.prev = i
				i.next = n.childItems.last.next
				n.childItems.last.next = i
				i.prev = n.childItems.last
				n.childItems.last = i

				/*
					println("checking new branch, insert")
					checkLevel(n, p)
					if len(p) > 0 {
						p = p[:len(p)-1]
					}
				*/

				continue
			}
		}

		/*
			println("checking unchanged")
			checkLevel(n, p)
			if len(p) > 0 {
				p = p[:len(p)-1]
			}
		*/

		find(n.childItems, i)
		break
	}

	/*
		println("checking root")
		if !checkComplete(t.root.items, true, false) {
			panic("add: root items not complete")
		}

		if !checkComplete(t.root.childItems, false, true) {
			panic("add: root child items not complete")
		}

		if !checkConnected(t.root.items, t.root.childItems) {
			panic("add: root items and child items not connected")
		}
	*/
}

func (t *tree) removeItem(i *item) {
	var (
		np []*node
		n  *node
		p  []string
	)

	if i.prev != nil {
		i.prev.next = i.next
	}

	if i.next != nil {
		i.next.prev = i.prev
	}

	np = t.nodePath(i.path)
	n = np[len(np)-1]
	if n.items.first == i {
		if n.items.last == i {
			n.items.first, n.items.last = nil, nil
		} else {
			n.items.first = i.next
		}
	} else {
		if n.items.last == i {
			n.items.last = i.prev
		}
	}

	if n.items.first == nil || n.items.last == nil {
		n.items.first, n.items.last = nil, nil
	}

	p = i.path
	for {
		if len(np) == 0 {
			break
		}

		n, np = np[len(np)-1], np[:len(np)-1]
		if n.childItems.first == i && n.childItems.last == i {
			n.childItems.first, n.childItems.last = nil, nil
		} else if n.childItems.first == i {
			n.childItems.first = i.next
		} else if n.childItems.last == i {
			n.childItems.last = i.prev
		}

		if n.childItems.first == nil || n.childItems.last == nil {
			n.childItems.first, n.childItems.last = nil, nil
		}

		if len(np) > 0 && n.items.first == nil && n.childItems.first == nil {
			delete(np[len(np)-1].children, p[len(p)-1])
		}

		if len(p) > 0 {
			p = p[:len(p)-1]
		}
	}

	/*
		if !checkComplete(t.root.items, true, false) {
			panic("remove: root items not complete")
		}

		if !checkComplete(t.root.childItems, false, true) {
			panic("remove: root child items not complete")
		}

		if !checkConnected(t.root.items, t.root.childItems) {
			panic("remove: root items and child items not connected")
		}
	*/
}
