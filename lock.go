package treelock

type lockType int

const (
	readLock lockType = iota
	writeLock
	treeReadLock
	treeWriteLock
)

type releaseLock func()

type Lock struct {
	root    *node
	tree    *tree
	notify  []*item
	acquire chan *item
	release chan *item
	quit    chan struct{}
}

func newItem(t lockType, path []string) *item {
	return &item{
		typ:    t,
		path:   path,
		notify: make(chan releaseLock),
	}
}

func New() *Lock {
	l := &Lock{
		root:    &node{},
		tree:    newTree(),
		acquire: make(chan *item),
		release: make(chan *item),
		quit:    make(chan struct{}),
	}

	go l.run()
	return l
}

func (l *Lock) getNodePath(path []string) []*node {
	np := []*node{l.root}
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

func (l *Lock) removeNode(path []string) {
	np := l.getNodePath(path)
	for {
		if len(np) == 1 {
			return
		}

		n, p := np[len(np)-1], np[len(np)-2]
		if len(n.nodeItems) > 0 {
			return
		}

		if len(n.children) > 0 {
			return
		}

		delete(p.children, path[len(path)-1])
		np, path = np[:len(np)-1], path[:len(path)-1]
	}
}

func getLockedNodes(n *node) []*node {
	var ln []*node
	if len(n.nodeItems) > 0 {
		ln = append(ln, n)
	}

	for _, c := range n.children {
		ln = append(ln, getLockedNodes(c)...)
	}

	return ln
}

func (l *Lock) doAcquire(i *item) {
	// println("/" + strings.Join(i.path, "/"))
	np := l.tree.nodePath(i.path)
	var blockedBy []*item
	for _, npn := range np[:len(np)-1] {
		npn.items.rangeOver(func(ni *item) {
			if ni.typ == treeWriteLock ||
				ni.typ == treeReadLock && (i.typ == treeWriteLock || i.typ == writeLock) {
				blockedBy = append(blockedBy, ni)
			}
		})
	}

	if i.typ == treeReadLock || i.typ == treeWriteLock {
		np[len(np)-1].childItems.rangeOver(func(ni *item) {
			if i.typ == treeWriteLock ||
				ni.typ == treeWriteLock ||
				ni.typ == writeLock {
				blockedBy = append(blockedBy, ni)
			}
		})
	}

	np[len(np)-1].items.rangeOver(func(ni *item) {
		if ni.typ == writeLock ||
			ni.typ == treeWriteLock ||
			i.typ == writeLock ||
			i.typ == treeWriteLock {
			blockedBy = append(blockedBy, ni)
		}
	})

	l.tree.addItem(i)
	i.blockedBy = blockedBy
	for _, b := range blockedBy {
		b.blocking = append(b.blocking, i)
	}

	if len(i.blockedBy) == 0 {
		l.notify = append(l.notify, i)
	}
}

func (l *Lock) doAcquire1(i *item) {
	np := l.getNodePath(i.path)
	n := np[len(np)-1]

	var blockedBy []*item
	for _, npn := range np[:len(np)-1] {
		for _, npi := range npn.nodeItems {
			if npi.typ == treeWriteLock ||
				npi.typ == treeReadLock && (i.typ == treeWriteLock || i.typ == writeLock) {
				blockedBy = append(blockedBy, npi)
			}
		}
	}

	if i.typ == treeReadLock || i.typ == treeWriteLock {
		ln := getLockedNodes(n)
		for _, lnn := range ln {
			if lnn == n {
				continue
			}

			for _, lni := range lnn.nodeItems {
				if i.typ == treeWriteLock ||
					lni.typ == treeWriteLock ||
					lni.typ == writeLock {
					blockedBy = append(blockedBy, lni)
				}
			}
		}
	}

	for _, ni := range n.nodeItems {
		if ni.typ == writeLock ||
			ni.typ == treeWriteLock ||
			i.typ == writeLock ||
			i.typ == treeWriteLock {
			blockedBy = append(blockedBy, ni)
		}
	}

	i.blockedBy = blockedBy
	for _, b := range blockedBy {
		b.blocking = append(b.blocking, i)
	}

	n.nodeItems = append(n.nodeItems, i)
	i.node = n
	if len(i.blockedBy) == 0 {
		l.notify = append(l.notify, i)
	}
}

func removeNodeItem(items []*item, item *item) []*item {
	for i := range items {
		if items[i] == item {
			return append(items[:i], items[i+1:]...)
		}
	}

	return items
}

func (l *Lock) doRelease(i *item) {
	l.tree.removeItem(i)
	for _, b := range i.blocking {
		b.blockedBy = removeNodeItem(b.blockedBy, i)
		if len(b.blockedBy) == 0 {
			l.notify = append(l.notify, b)
		}
	}
}

func (l *Lock) doRelease1(i *item) {
	i.node.nodeItems = removeNodeItem(i.node.nodeItems, i)
	l.removeNode(i.path)
	for _, b := range i.blocking {
		b.blockedBy = removeNodeItem(b.blockedBy, i)
		if len(b.blockedBy) == 0 {
			l.notify = append(l.notify, b)
		}
	}
}

func (l *Lock) notifyNext() (chan<- releaseLock, releaseLock) {
	if len(l.notify) == 0 {
		return nil, nil
	}

	item := l.notify[0]
	l.notify = l.notify[1:]
	release := func() {
		select {
		case l.release <- item:
		case <-l.quit:
		}
	}

	return item.notify, release
}

func (l *Lock) run() {
	var (
		notify  chan<- releaseLock
		release releaseLock
	)

	for {
		if notify == nil {
			notify, release = l.notifyNext()
		}

		select {
		case a := <-l.acquire:
			l.doAcquire(a)
		case r := <-l.release:
			l.doRelease(r)
		case notify <- release:
			notify, release = nil, nil
		case <-l.quit:
			return
		}
	}
}

func (l *Lock) requestLock(typ lockType, path []string) releaseLock {
	i := newItem(typ, path)
	select {
	case l.acquire <- i:
	case <-l.quit:
		return func() {}
	}

	select {
	case release := <-i.notify:
		return release
	case <-l.quit:
		return func() {}
	}
}

func (l *Lock) ReadNode(path ...string) releaseLock  { return l.requestLock(readLock, path) }
func (l *Lock) WriteNode(path ...string) releaseLock { return l.requestLock(writeLock, path) }
func (l *Lock) ReadTree(path ...string) releaseLock  { return l.requestLock(treeReadLock, path) }
func (l *Lock) WriteTree(path ...string) releaseLock { return l.requestLock(treeWriteLock, path) }
func (l *Lock) Close()                               { close(l.quit) }
