package treelock

import "sync"

type lockType int

const (
	readLock lockType = iota
	writeLock
	treeReadLock
	treeWriteLock
)

type releaseLock func()

type item struct {
	typ       lockType
	path      []string
	element   *element
	blockedBy *sync.WaitGroup
	blocking  []*item
}

type Lock struct {
	tree *tree
	mx   *sync.Mutex
}

func newItem(t lockType, path []string) *item {
	return &item{
		typ:       t,
		path:      path,
		blockedBy: &sync.WaitGroup{},
	}
}

func New() *Lock {
	return &Lock{
		tree: newTree(),
		mx:   &sync.Mutex{},
	}
}

func (l *Lock) doAcquire(i *item) {
	l.mx.Lock()
	defer l.mx.Unlock()
	np := l.tree.nodePath(i.path)
	var blockedBy []*item
	for _, npn := range np[:len(np)-1] {
		npn.items.rangeOver(func(ni *item) {
			if ni.typ == treeWriteLock ||
				ni.typ == treeReadLock &&
					(i.typ == treeWriteLock || i.typ == writeLock) {
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

	if i.typ == treeReadLock || i.typ == treeWriteLock {
		np[len(np)-1].subtreeItems.rangeOver(func(ni *item) {
			if i.typ == treeWriteLock ||
				ni.typ == treeWriteLock ||
				ni.typ == writeLock {
				blockedBy = append(blockedBy, ni)
			}
		})
	}

	l.tree.insert(np, i)
	i.blockedBy.Add(len(blockedBy))
	for _, b := range blockedBy {
		b.blocking = append(b.blocking, i)
	}
}

func (l *Lock) doRelease(i *item) {
	l.mx.Lock()
	defer l.mx.Unlock()
	l.tree.remove(i)
	for _, b := range i.blocking {
		b.blockedBy.Done()
	}
}

func (l *Lock) requestLock(typ lockType, path []string) releaseLock {
	i := newItem(typ, path)
	l.doAcquire(i)
	i.blockedBy.Wait()
	return func() {
		l.doRelease(i)
	}
}

func (l *Lock) ReadNode(path ...string) releaseLock  { return l.requestLock(readLock, path) }
func (l *Lock) WriteNode(path ...string) releaseLock { return l.requestLock(writeLock, path) }
func (l *Lock) ReadTree(path ...string) releaseLock  { return l.requestLock(treeReadLock, path) }
func (l *Lock) WriteTree(path ...string) releaseLock { return l.requestLock(treeWriteLock, path) }
