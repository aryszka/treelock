package treelock

import "sync"

type lockType int

const (
	readLock lockType = iota
	writeLock
	treeReadLock
	treeWriteLock
)

type operation struct {
	typ       lockType
	path      []string
	item      *item
	blockedBy *sync.WaitGroup
	blocking  []*operation
}

type Lock struct {
	tree *tree
	mx   *sync.Mutex
}

func newOperation(t lockType, path []string) *operation {
	return &operation{
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

func (l *Lock) doAcquire(o *operation) {
	l.mx.Lock()
	defer l.mx.Unlock()
	np := l.tree.nodePath(o.path)
	var blockedBy []*operation
	for _, npn := range np[:len(np)-1] {
		npn.operations.rangeOver(func(no *operation) {
			if no.typ == treeWriteLock ||
				no.typ == treeReadLock &&
					(o.typ == treeWriteLock || o.typ == writeLock) {
				blockedBy = append(blockedBy, no)
			}
		})
	}

	np[len(np)-1].operations.rangeOver(func(no *operation) {
		if no.typ == writeLock ||
			no.typ == treeWriteLock ||
			o.typ == writeLock ||
			o.typ == treeWriteLock {
			blockedBy = append(blockedBy, no)
		}
	})

	if o.typ == treeReadLock || o.typ == treeWriteLock {
		np[len(np)-1].subtreeOperations.rangeOver(func(no *operation) {
			if o.typ == treeWriteLock ||
				no.typ == treeWriteLock ||
				no.typ == writeLock {
				blockedBy = append(blockedBy, no)
			}
		})
	}

	l.tree.insert(np, o)
	o.blockedBy.Add(len(blockedBy))
	for _, b := range blockedBy {
		b.blocking = append(b.blocking, o)
	}
}

func (l *Lock) release(o *operation) {
	l.mx.Lock()
	defer l.mx.Unlock()
	l.tree.remove(o)
	for _, b := range o.blocking {
		b.blockedBy.Done()
	}
}

func (l *Lock) acquire(typ lockType, path []string) func() {
	o := newOperation(typ, path)
	l.doAcquire(o)
	o.blockedBy.Wait()
	return func() {
		l.release(o)
	}
}

func (l *Lock) ReadNode(path ...string) func()  { return l.acquire(readLock, path) }
func (l *Lock) WriteNode(path ...string) func() { return l.acquire(writeLock, path) }
func (l *Lock) ReadTree(path ...string) func()  { return l.acquire(treeReadLock, path) }
func (l *Lock) WriteTree(path ...string) func() { return l.acquire(treeWriteLock, path) }
