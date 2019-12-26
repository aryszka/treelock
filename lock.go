package treelock

type lockType int

const (
	readLock lockType = iota
	writeLock
	treeReadLock
	treeWriteLock
)

type releaseLock func()

type item struct {
	typ    lockType
	path   []string
	notify chan releaseLock
}

type node struct {
	children       map[string]*node
	readingNode    int
	writingNode    bool
	readingBranch  int
	writingBranch  int
	readingSubtree int
	writingSubtree bool
}

type Lock struct {
	root             *node
	waiting          []*item
	acquire, release chan *item
	quit             chan struct{}
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

func (l *Lock) cleanup(path []string) {
	np := l.getNodePath(path)
	for {
		if len(np) == 1 {
			return
		}

		n, p := np[len(np)-1], np[len(np)-2]
		if n.readingNode > 0 || n.writingNode ||
			n.readingBranch > 0 || n.writingBranch > 0 ||
			n.readingSubtree > 0 || n.writingSubtree {
			return
		}

		if len(n.children) > 0 {
			return
		}

		delete(p.children, path[len(path)-1])
		np, path = np[:len(np)-1], path[:len(path)-1]
	}
}

func (l *Lock) notifyNext() (chan<- releaseLock, releaseLock) {
	if len(l.waiting) == 0 {
		return nil, nil
	}

	next := l.waiting[0]
	np := l.getNodePath(next.path)
	switch next.typ {
	case readLock:
		n := np[len(np)-1]
		if n.writingNode {
			return nil, nil
		}

		var p *node
		pp := np
		for len(pp) > 0 {
			p, pp = pp[len(pp)-1], pp[:len(pp)-1]
			if p.writingSubtree {
				return nil, nil
			}
		}

		n.readingNode++
		pp = np
		for len(pp) > 0 {
			p, pp = pp[len(pp)-1], pp[:len(pp)-1]
			p.readingBranch++
		}
	case writeLock:
		n := np[len(np)-1]
		if n.writingNode || n.readingNode > 0 {
			return nil, nil
		}

		var p *node
		pp := np
		for len(pp) > 0 {
			p, pp = pp[len(pp)-1], pp[:len(pp)-1]
			if p.writingSubtree || p.readingSubtree > 0 {
				return nil, nil
			}
		}

		n.writingNode = true
		pp = np
		for len(pp) > 0 {
			p, pp = pp[len(pp)-1], pp[:len(pp)-1]
			p.writingBranch++
		}
	case treeReadLock:
		n := np[len(np)-1]
		if n.writingBranch > 0 {
			return nil, nil
		}

		var p *node
		pp := np
		for len(pp) > 0 {
			p, pp = pp[len(pp)-1], pp[:len(pp)-1]
			if p.writingSubtree {
				return nil, nil
			}
		}

		n.readingSubtree++
		pp = np
		for len(pp) > 0 {
			p, pp = pp[len(pp)-1], pp[:len(pp)-1]
			p.readingBranch++
		}
	case treeWriteLock:
		n := np[len(np)-1]
		if n.writingBranch > 0 || n.readingBranch > 0 {
			return nil, nil
		}

		var p *node
		pp := np
		for len(pp) > 0 {
			p, pp = pp[len(pp)-1], pp[:len(pp)-1]
			if p.writingSubtree || p.readingSubtree > 0 {
				return nil, nil
			}
		}

		n.writingSubtree = true
		pp = np
		for len(pp) > 0 {
			p, pp = pp[len(pp)-1], pp[:len(pp)-1]
			p.writingBranch++
		}
	}

	l.waiting = l.waiting[1:]
	return next.notify, func() { l.release <- next }
}

func (l *Lock) doAcquire(i *item) {
	l.waiting = append(l.waiting, i)
}

func (l *Lock) doRelease(i *item) {
	nn := l.getNodePath(i.path)
	switch i.typ {
	case readLock:
		n := nn[len(nn)-1]
		n.readingNode--
		pp := nn
		var p *node
		for len(pp) > 0 {
			p, pp = pp[len(pp)-1], pp[:len(pp)-1]
			p.readingBranch--
		}
	case writeLock:
		n := nn[len(nn)-1]
		n.writingNode = false
		pp := nn
		var p *node
		for len(pp) > 0 {
			p, pp = pp[len(pp)-1], pp[:len(pp)-1]
			p.writingBranch--
		}
	case treeReadLock:
		n := nn[len(nn)-1]
		n.readingSubtree--
		pp := nn
		var p *node
		for len(pp) > 0 {
			p, pp = pp[len(pp)-1], pp[:len(pp)-1]
			p.readingBranch--
		}
	case treeWriteLock:
		n := nn[len(nn)-1]
		n.writingSubtree = false
		pp := nn
		var p *node
		for len(pp) > 0 {
			p, pp = pp[len(pp)-1], pp[:len(pp)-1]
			p.writingBranch--
		}
	}

	l.cleanup(i.path)
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
			// TODO: cleanup
			return
		}
	}
}

func (l *Lock) ReadNode(path ...string) releaseLock {
	i := newItem(readLock, path)
	l.acquire <- i
	return <-i.notify
}

func (l *Lock) WriteNode(path ...string) releaseLock {
	i := newItem(writeLock, path)
	l.acquire <- i
	return <-i.notify
}

func (l *Lock) ReadTree(path ...string) releaseLock {
	i := newItem(treeReadLock, path)
	l.acquire <- i
	return <-i.notify
}

func (l *Lock) WriteTree(path ...string) releaseLock {
	i := newItem(treeWriteLock, path)
	l.acquire <- i
	return <-i.notify
}

func (l *Lock) Close() {
	close(l.quit)
}
