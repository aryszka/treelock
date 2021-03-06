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
	blockedBy sync.WaitGroup
	blocking  []*operation
}

// L instances provide read/write locking for tree structures with
// nodes referenced by their path.
type L struct {
	tree *node
	mx   sync.Mutex
}

func blockedByOnPath(o *operation, nodePath []*node) []*operation {
	var ops []*operation
	for _, n := range nodePath {
		rangeOver(n.operations, func(no *operation) {
			if no.typ == treeWriteLock ||
				no.typ == treeReadLock &&
					(o.typ == treeWriteLock || o.typ == writeLock) {
				ops = append(ops, no)
			}
		})
	}

	return ops
}

func blockedByOnNode(o *operation, n *node) []*operation {
	var ops []*operation
	rangeOver(n.operations, func(no *operation) {
		if no.typ == writeLock ||
			no.typ == treeWriteLock ||
			o.typ == writeLock ||
			o.typ == treeWriteLock {
			ops = append(ops, no)
		}
	})

	return ops
}

func blockedByOnSubtree(o *operation, n *node) []*operation {
	var ops []*operation
	rangeOver(n.subtreeOperations, func(no *operation) {
		if o.typ == treeWriteLock ||
			no.typ == treeWriteLock ||
			no.typ == writeLock {
			ops = append(ops, no)
		}
	})

	return ops
}

func initBlocking(nodePath []*node, o *operation) {
	var blockedBy []*operation
	n, np := nodePath[len(nodePath)-1], nodePath[:len(nodePath)-1]
	blockedBy = append(blockedBy, blockedByOnPath(o, np)...)
	blockedBy = append(blockedBy, blockedByOnNode(o, n)...)
	if o.typ == treeReadLock || o.typ == treeWriteLock {
		blockedBy = append(blockedBy, blockedByOnSubtree(o, n)...)
	}

	o.blockedBy.Add(len(blockedBy))
	for _, b := range blockedBy {
		b.blocking = append(b.blocking, o)
	}
}

func (l *L) acquire(typ lockType, path []string) func() {
	o := &operation{
		typ:  typ,
		path: path,
	}

	l.mx.Lock()
	if l.tree == nil {
		l.tree = &node{}
	}

	np := nodePath(l.tree, o.path)
	initBlocking(np, o)
	insert(np, o)
	l.mx.Unlock()
	o.blockedBy.Wait()
	return func() {
		l.release(o)
	}
}

func (l *L) release(o *operation) {
	l.mx.Lock()
	defer l.mx.Unlock()
	np := nodePath(l.tree, o.path)
	remove(np, o)
	for _, b := range o.blocking {
		b.blockedBy.Done()
	}
}

// ReadNode acquires a read lock for an individual node represented by
// its path. It blocks until no preceding operations hold a write lock
// preventing the read from this node. The returned function must be
// called to release the lock when the operation finished.
//
// While holding the lock, subsequent operations will be blocked if they
// try to acquire a write lock on the current node, or a write tree lock
// on the path to the current node.
//
func (l *L) ReadNode(path ...string) func() {
	return l.acquire(readLock, path)
}

// WriteNode acquires a write lock for an individual node represented by
// its path. It blocks until no preceding operations hold any locks
// preventing the write to this node. The returned function must be
// called to release the lock when the operation finished.
//
// While holding the lock, subsequent operations will be blocked if they
// try to acquire a read or write lock on the current node, or a read or
// write tree lock on the path to the current node.
//
func (l *L) WriteNode(path ...string) func() {
	return l.acquire(writeLock, path)
}

// ReadTree acquires a read lock for the subtree starting from the node
// represented by the current path. It blocks until no preceding
// operations hold a write lock preventing reads from the nodes in the
// subtree. The returned function must be called to release the lock
// when the operation finished.
//
// While holding the lock, subsequent operations will be blocked if they
// try to acquire a write lock in the subtree, including the current
// node, or a write tree lock on the path to the current node.
//
func (l *L) ReadTree(path ...string) func() {
	return l.acquire(treeReadLock, path)
}

// WriteTree acquires a write lock for the subtree starting from the
// node represented by the current path. It blocks until no preceding
// operations hold any locks preventing writes or structural changes to
// the nodes in the subtree. The returned function must be called to
// release the lock when the operation finished.
//
// While holding the lock, subsequent operations will be blocked if they
// try to acquire any locks in the subtree, including the current node,
// or a read or write tree lock on the path to the current node.
//
func (l *L) WriteTree(path ...string) func() {
	return l.acquire(treeWriteLock, path)
}
