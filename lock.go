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
	tree    *tree
	notify  list
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
		tree:    newTree(),
		acquire: make(chan *item),
		release: make(chan *item),
		quit:    make(chan struct{}),
	}

	go l.run()
	return l
}

func (l *Lock) doAcquire(i *item) {
	np := l.tree.nodePath(i.path)
	var blockedBy []*item
	for _, npn := range np[:len(np)-1] {
		npn.items.rangeOver(func(ne *element) {
			if ne.item.typ == treeWriteLock ||
				ne.item.typ == treeReadLock && (i.typ == treeWriteLock || i.typ == writeLock) {
				blockedBy = append(blockedBy, ne.item)
			}
		})
	}

	if i.typ == treeReadLock || i.typ == treeWriteLock {
		np[len(np)-1].subtreeItems.rangeOver(func(ne *element) {
			if i.typ == treeWriteLock ||
				ne.item.typ == treeWriteLock ||
				ne.item.typ == writeLock {
				blockedBy = append(blockedBy, ne.item)
			}
		})
	}

	np[len(np)-1].items.rangeOver(func(ne *element) {
		if ne.item.typ == writeLock ||
			ne.item.typ == treeWriteLock ||
			i.typ == writeLock ||
			i.typ == treeWriteLock {
			blockedBy = append(blockedBy, ne.item)
		}
	})

	e := &element{item: i}
	i.element = e
	l.tree.addElement(e)
	i.blockedBy = len(blockedBy)
	for _, b := range blockedBy {
		b.blocking = append(b.blocking, i)
	}

	if i.blockedBy == 0 {
		l.notify = l.notify.insert(&element{item: i})
	}
}

func (l *Lock) doRelease(i *item) {
	l.tree.removeElement(i.element)
	for _, b := range i.blocking {
		b.blockedBy--
		if b.blockedBy == 0 {
			l.notify = l.notify.insert(&element{item: b})
		}
	}
}

func (l *Lock) notifyNext() (chan<- releaseLock, releaseLock) {
	if l.notify.first == nil {
		return nil, nil
	}

	first := l.notify.first
	l.notify = l.notify.remove(first)
	item := first.item
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
