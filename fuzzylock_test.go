package treelock

import (
	"math/rand"
	"sync"
	"testing"
	"time"
)

const (
	fuzzyDuration      = 3 * time.Second
	shortFuzzyDuration = time.Second
	fuzzyConcurrency   = 2048
)

type counter chan int

type intRange struct {
	min, max int
}

type timeRange struct {
	min, max time.Duration
}

type testNode struct {
	children map[string]*testNode
	writing  bool
}

type locker interface {
	ReadNode(...string) func()
	WriteNode(...string) func()
	ReadTree(...string) func()
	WriteTree(...string) func()
}

type testLock struct {
	mx *sync.RWMutex
}

var (
	busyDuration    = timeRange{300 * time.Microsecond, 1200 * time.Microsecond}
	fuzzyTreeLevels = 6
	firstLevelNodes = intRange{3, 5}
	cnt             = newCounter()
)

func newCounter() counter {
	c := make(chan int, 1)
	c <- 0
	return c
}

func (c counter) inc() {
	v := <-c
	v++
	c <- v
}

func (c counter) value() int {
	v := <-c
	c <- v
	return v
}

func randomInt(r intRange) int {
	return r.min + rand.Intn(r.max-r.min)
}

func randomDuration(r timeRange) time.Duration {
	i := randomInt(intRange{int(r.min), int(r.max)})
	return time.Duration(i)
}

func (n *testNode) get(path ...string) *testNode {
	if len(path) == 0 {
		return n
	}

	if n = n.children[path[0]]; n == nil {
		panic("invalid path")
	}

	return n.get(path[1:]...)
}

func (l *testLock) ReadNode(...string) func() {
	l.mx.RLock()
	return func() { l.mx.RUnlock() }
}

func (l *testLock) WriteNode(...string) func() {
	l.mx.Lock()
	return func() { l.mx.Unlock() }
}

func (l *testLock) ReadTree(...string) func() {
	return l.ReadNode()
}

func (l *testLock) WriteTree(...string) func() {
	return l.WriteNode()
}

func testAccess(t *testing.T, tree *testNode, lockMethod func(...string) func(), path []string, write bool) {
	defer lockMethod(path...)()
	n := tree.get(path...)
	if n.writing {
		t.Error("busy node found")
	}

	if write {
		n.writing = true
	}

	time.Sleep(randomDuration(busyDuration))
	if write {
		if !n.writing {
			t.Error("busy value set concurrently")
		}

		n.writing = false
	}

	cnt.inc()
}

func selectMethod(l locker) (func(...string) func(), bool) {
	readMethods := []func(...string) func(){
		l.ReadNode,
		l.ReadTree,
	}

	writeMethods := []func(...string) func(){
		l.WriteNode,
		l.WriteTree,
	}

	i := randomInt(intRange{0, len(readMethods) + len(writeMethods)})
	write := i >= len(readMethods)
	if write {
		return writeMethods[i-len(readMethods)], true
	}

	return readMethods[i], false
}

func selectPath(paths [][][]string) []string {
	g := randomInt(intRange{0, len(paths)})
	i := randomInt(intRange{0, len(paths[g])})
	return paths[g][i]
}

func callTestAccess(t *testing.T, tree *testNode, l locker, paths [][][]string) {
	method, write := selectMethod(l)
	path := selectPath(paths)
	testAccess(t, tree, method, path, write)
}

func testLoop(t *testing.T, timeout <-chan struct{}, tree *testNode, l locker, paths [][][]string) {
	for {
		select {
		case <-timeout:
			return
		default:
			callTestAccess(t, tree, l, paths)
		}
	}
}

func buildTree() *testNode {
	var createChildren func(int, *testNode)
	createChildren = func(level int, n *testNode) {
		if level == fuzzyTreeLevels {
			return
		}

		n.children = make(map[string]*testNode)
		nodes := randomInt(intRange{
			firstLevelNodes.min * level,
			firstLevelNodes.max * level,
		})

		for i := 0; i < nodes; i++ {
			c := &testNode{}
			n.children[string('a'+i)] = c
			createChildren(level+1, c)
		}
	}

	root := &testNode{}
	createChildren(1, root)
	return root
}

func getAllPaths(node *testNode) [][]string {
	paths := [][]string{nil}
	for pathElement, c := range node.children {
		p := getAllPaths(c)
		for i := range p {
			p[i] = append([]string{pathElement}, p[i]...)
		}

		paths = append(paths, p...)
	}

	return paths
}

func groupPaths(paths [][]string) [][][]string {
	groups := make(map[int][][]string)
	for _, p := range paths {
		groups[len(p)] = append(groups[len(p)], p)
	}

	gi := make(map[int]int)
	var i int
	for l := range groups {
		gi[l] = i
		i++
	}

	gpaths := make([][][]string, len(groups))
	for l, i := range gi {
		gpaths[i] = groups[l]
	}

	return gpaths
}

func testLockFuzzy(t *testing.T, d time.Duration) {
	before := cnt.value()
	l := New()
	// l := &testLock{mx: &sync.RWMutex{}}
	tree := buildTree()
	paths := getAllPaths(tree)
	gpaths := groupPaths(paths)
	timeout := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < fuzzyConcurrency; i++ {
		wg.Add(1)
		go func() {
			testLoop(t, timeout, tree, l, gpaths)
			wg.Done()
		}()
	}

	done := make(chan struct{})
	go func() {
		<-time.After(d)
		close(timeout)
		select {
		case <-time.After(3 * d):
			panic("fuzzy test did not complete")
		case <-done:
		}
	}()

	wg.Wait()
	close(done)
	t.Log("access", cnt.value()-before)
}

func TestLockFuzzy(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	testLockFuzzy(t, fuzzyDuration)
}

func TestLockFuzzyShort(t *testing.T) {
	if testing.CoverMode() != "" || !testing.Short() {
		t.Skip()
	}

	testLockFuzzy(t, shortFuzzyDuration)
}
