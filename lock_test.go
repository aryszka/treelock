package treelock

import (
	"testing"
	"time"
)

const (
	minDelay        = 9 * time.Millisecond
	maxTestDuration = time.Second
)

func testLocked(t *testing.T, l *L, release func(), method func(...string) func(), path ...string) {
	released := make(chan struct{})
	access := func() <-chan struct{} {
		done := make(chan struct{})
		go func() {
			r := method(path...)
			select {
			case <-released:
			default:
				t.Error("acquired before released")
			}

			r()
			close(done)
		}()

		return done
	}

	done1 := access()
	done2 := access()
	time.Sleep(minDelay)
	close(released)
	release()
	<-done1
	<-done2
}

func testRun(t *testing.T, name string, test func(*testing.T)) {
	t.Run(name, func(t *testing.T) {
		done := make(chan struct{})
		timeout := time.After(maxTestDuration)
		go func() {
			select {
			case <-done:
			case <-timeout:
				panic("test did not complete: " + t.Name())
			}
		}()

		test(t)
		close(done)
	})
}

func TestLockRead(t *testing.T) {
	t.Run("root", func(t *testing.T) {
		testRun(t, "single", func(t *testing.T) {
			l := new(L)
			r := l.ReadNode()
			r()
		})

		testRun(t, "multiple", func(t *testing.T) {
			l := new(L)
			r1 := l.ReadNode()
			r2 := l.ReadNode()
			r1()
			r2()
		})
	})

	t.Run("non-root", func(t *testing.T) {
		testRun(t, "single", func(t *testing.T) {
			l := new(L)
			r := l.ReadNode("foo", "bar")
			r()
		})

		testRun(t, "multiple", func(t *testing.T) {
			l := new(L)
			r1 := l.ReadNode("foo", "bar")
			r2 := l.ReadNode("foo", "bar")
			r1()
			r2()
		})
	})

	testRun(t, "parent locked", func(t *testing.T) {
		l := new(L)
		r1 := l.ReadNode("foo")
		r2 := l.ReadNode("foo", "bar")
		r2()
		r1()
	})

	testRun(t, "sibling locked", func(t *testing.T) {
		l := new(L)
		r1 := l.ReadNode("foo", "bar")
		r2 := l.ReadNode("foo", "baz")
		r2()
		r1()
	})

	testRun(t, "child locked", func(t *testing.T) {
		l := new(L)
		r1 := l.ReadNode("foo", "bar", "baz")
		r2 := l.ReadNode("foo", "bar")
		r2()
		r1()
	})

	testRun(t, "has child and siblings", func(t *testing.T) {
		l := new(L)
		r1 := l.ReadNode("a")
		r2 := l.ReadNode("b", "a")
		r3 := l.ReadNode("c")
		r4 := l.ReadNode("b")
		r4()
		r3()
		r2()
		r1()
	})

	testRun(t, "has lock and siblings", func(t *testing.T) {
		l := new(L)
		r1 := l.ReadNode("a")
		r2 := l.ReadNode("b")
		r3 := l.ReadNode("c")
		r4 := l.ReadNode("b")
		r4()
		r3()
		r2()
		r1()
	})

	testRun(t, "parent has item and sibling", func(t *testing.T) {
		l := new(L)
		r1 := l.ReadNode("a")
		r2 := l.ReadNode("b")
		r3 := l.ReadNode("a", "a")
		r3()
		r2()
		r1()
	})

	testRun(t, "parent has children and sibling", func(t *testing.T) {
		l := new(L)
		r1 := l.ReadNode("a", "a")
		r2 := l.ReadNode("b")
		r3 := l.ReadNode("a", "b")
		r3()
		r2()
		r1()
	})
}

func TestLockWrite(t *testing.T) {
	t.Run("root", func(t *testing.T) {
		testRun(t, "unlocked", func(t *testing.T) {
			l := new(L)
			r := l.WriteNode()
			r()
		})

		testRun(t, "locked", func(t *testing.T) {
			l := new(L)
			r := l.WriteNode()
			testLocked(t, l, r, l.WriteNode)
		})
	})

	t.Run("non-root", func(t *testing.T) {
		testRun(t, "unlocked", func(t *testing.T) {
			l := new(L)
			r := l.WriteNode("foo", "bar")
			r()
		})

		testRun(t, "locked", func(t *testing.T) {
			l := new(L)
			r := l.WriteNode("foo", "bar")
			testLocked(t, l, r, l.WriteNode, "foo", "bar")
		})
	})

	testRun(t, "parent locked", func(t *testing.T) {
		l := new(L)
		r1 := l.WriteNode("foo")
		r2 := l.WriteNode("foo", "bar")
		r2()
		r1()
	})

	testRun(t, "sibling locked", func(t *testing.T) {
		l := new(L)
		r1 := l.WriteNode("foo", "bar")
		r2 := l.WriteNode("foo", "baz")
		r2()
		r1()
	})

	testRun(t, "child locked", func(t *testing.T) {
		l := new(L)
		r1 := l.WriteNode("foo", "bar", "baz")
		r2 := l.WriteNode("foo", "bar")
		r2()
		r1()
	})
}

func TestLockReadWrite(t *testing.T) {
	t.Run("root", func(t *testing.T) {
		testRun(t, "read and write", func(t *testing.T) {
			l := new(L)
			r := l.ReadNode()
			testLocked(t, l, r, l.WriteNode)
		})

		testRun(t, "write and read", func(t *testing.T) {
			l := new(L)
			r := l.WriteNode()
			testLocked(t, l, r, l.ReadNode)
		})
	})

	t.Run("non-root", func(t *testing.T) {
		testRun(t, "read and write", func(t *testing.T) {
			l := new(L)
			r := l.ReadNode("foo", "bar")
			testLocked(t, l, r, l.WriteNode, "foo", "bar")
		})

		testRun(t, "write and read", func(t *testing.T) {
			l := new(L)
			r := l.WriteNode("foo", "bar")
			testLocked(t, l, r, l.ReadNode, "foo", "bar")
		})
	})

	t.Run("parent locked", func(t *testing.T) {
		testRun(t, "read and write", func(t *testing.T) {
			l := new(L)
			r1 := l.ReadNode("foo")
			r2 := l.WriteNode("foo", "bar")
			r2()
			r1()
		})

		testRun(t, "write and read", func(t *testing.T) {
			l := new(L)
			r1 := l.WriteNode("foo")
			r2 := l.ReadNode("foo", "bar")
			r2()
			r1()
		})
	})

	t.Run("sibling locked", func(t *testing.T) {
		testRun(t, "read and write", func(t *testing.T) {
			l := new(L)
			r1 := l.ReadNode("foo", "bar")
			r2 := l.WriteNode("foo", "baz")
			r2()
			r1()
		})

		testRun(t, "write and read", func(t *testing.T) {
			l := new(L)
			r1 := l.WriteNode("foo", "bar")
			r2 := l.ReadNode("foo", "baz")
			r2()
			r1()
		})
	})

	t.Run("child locked", func(t *testing.T) {
		testRun(t, "read and write", func(t *testing.T) {
			l := new(L)
			r1 := l.ReadNode("foo", "bar", "baz")
			r2 := l.WriteNode("foo", "bar")
			r2()
			r1()
		})

		testRun(t, "write and read", func(t *testing.T) {
			l := new(L)
			r1 := l.WriteNode("foo", "bar", "baz")
			r2 := l.ReadNode("foo", "bar")
			r2()
			r1()
		})
	})
}

func TestLockReadTree(t *testing.T) {
	t.Run("root", func(t *testing.T) {
		testRun(t, "single", func(t *testing.T) {
			l := new(L)
			r := l.ReadTree()
			r()
		})

		testRun(t, "multiple", func(t *testing.T) {
			l := new(L)
			r1 := l.ReadTree()
			r2 := l.ReadTree()
			r2()
			r1()
		})
	})

	t.Run("non-root", func(t *testing.T) {
		testRun(t, "single", func(t *testing.T) {
			l := new(L)
			r := l.ReadTree("foo", "bar")
			r()
		})

		testRun(t, "multiple", func(t *testing.T) {
			l := new(L)
			r1 := l.ReadTree("foo", "bar")
			r2 := l.ReadTree("foo", "bar")
			r2()
			r1()
		})
	})

	t.Run("parent locked", func(t *testing.T) {
		testRun(t, "root", func(t *testing.T) {
			l := new(L)
			r1 := l.ReadTree()
			r2 := l.ReadTree("foo", "bar")
			r2()
			r1()
		})

		testRun(t, "non-root", func(t *testing.T) {
			l := new(L)
			r1 := l.ReadTree("foo", "bar")
			r2 := l.ReadTree("foo", "bar", "baz")
			r2()
			r1()
		})
	})

	testRun(t, "sibling locked", func(t *testing.T) {
		l := new(L)
		r1 := l.ReadTree("foo", "bar")
		r2 := l.ReadTree("foo", "baz")
		r2()
		r1()
	})

	t.Run("child locked", func(t *testing.T) {
		testRun(t, "root", func(t *testing.T) {
			l := new(L)
			r1 := l.ReadTree("foo", "bar")
			r2 := l.ReadTree()
			r2()
			r1()
		})

		testRun(t, "non-root", func(t *testing.T) {
			l := new(L)
			r1 := l.ReadTree("foo", "bar", "baz")
			r2 := l.ReadTree("foo", "bar")
			r2()
			r1()
		})
	})
}

func TestLockReadReadTree(t *testing.T) {
	t.Run("root", func(t *testing.T) {
		testRun(t, "read and read tree", func(t *testing.T) {
			l := new(L)
			r1 := l.ReadNode()
			r2 := l.ReadTree()
			r2()
			r1()
		})

		testRun(t, "read tree and read", func(t *testing.T) {
			l := new(L)
			r1 := l.ReadTree()
			r2 := l.ReadNode()
			r2()
			r1()
		})
	})

	t.Run("non-root", func(t *testing.T) {
		testRun(t, "read and read tree", func(t *testing.T) {
			l := new(L)
			r1 := l.ReadNode("foo", "bar")
			r2 := l.ReadTree("foo", "bar")
			r2()
			r1()
		})

		testRun(t, "read tree and read", func(t *testing.T) {
			l := new(L)
			r1 := l.ReadTree("foo", "bar")
			r2 := l.ReadNode("foo", "bar")
			r2()
			r1()
		})
	})

	t.Run("parent locked", func(t *testing.T) {
		testRun(t, "read and read tree", func(t *testing.T) {
			l := new(L)
			r1 := l.ReadNode("foo", "bar")
			r2 := l.ReadTree("foo", "bar", "baz")
			r2()
			r1()
		})

		testRun(t, "read tree and read", func(t *testing.T) {
			l := new(L)
			r1 := l.ReadTree("foo", "bar")
			r2 := l.ReadNode("foo", "bar", "baz")
			r2()
			r1()
		})
	})

	t.Run("sibling locked", func(t *testing.T) {
		testRun(t, "read and read tree", func(t *testing.T) {
			l := new(L)
			r1 := l.ReadNode("foo", "bar")
			r2 := l.ReadTree("foo", "baz")
			r2()
			r1()
		})

		testRun(t, "read tree and read", func(t *testing.T) {
			l := new(L)
			r1 := l.ReadTree("foo", "bar")
			r2 := l.ReadNode("foo", "baz")
			r2()
			r1()
		})
	})

	t.Run("child locked", func(t *testing.T) {
		testRun(t, "read and read tree", func(t *testing.T) {
			l := new(L)
			r1 := l.ReadNode("foo", "bar", "baz")
			r2 := l.ReadTree("foo", "bar")
			r2()
			r1()
		})

		testRun(t, "read tree and read", func(t *testing.T) {
			l := new(L)
			r1 := l.ReadTree("foo", "bar", "baz")
			r2 := l.ReadNode("foo", "bar")
			r2()
			r1()
		})
	})
}

func TestLockWriteReadTree(t *testing.T) {
	t.Run("root", func(t *testing.T) {
		testRun(t, "write and read tree", func(t *testing.T) {
			l := new(L)
			r := l.WriteNode()
			testLocked(t, l, r, l.ReadTree)
		})

		testRun(t, "read tree and write", func(t *testing.T) {
			l := new(L)
			r := l.ReadTree()
			testLocked(t, l, r, l.WriteNode)
		})
	})

	t.Run("non-root", func(t *testing.T) {
		testRun(t, "write and read tree", func(t *testing.T) {
			l := new(L)
			r := l.WriteNode("foo", "bar")
			testLocked(t, l, r, l.ReadTree, "foo", "bar")
		})

		testRun(t, "read tree and write", func(t *testing.T) {
			l := new(L)
			r := l.ReadTree("foo", "bar")
			testLocked(t, l, r, l.WriteNode, "foo", "bar")
		})
	})

	t.Run("parent locked", func(t *testing.T) {
		testRun(t, "write and read tree", func(t *testing.T) {
			l := new(L)
			r1 := l.WriteNode("foo", "bar")
			r2 := l.ReadTree("foo", "bar", "baz")
			r2()
			r1()
		})

		testRun(t, "read tree and write", func(t *testing.T) {
			l := new(L)
			r := l.ReadTree("foo", "bar")
			testLocked(t, l, r, l.WriteNode, "foo", "bar", "baz")
		})
	})

	t.Run("sibling locked", func(t *testing.T) {
		testRun(t, "write and read tree", func(t *testing.T) {
			l := new(L)
			r1 := l.WriteNode("foo", "bar")
			r2 := l.ReadTree("foo", "baz")
			r2()
			r1()
		})

		testRun(t, "read tree and write", func(t *testing.T) {
			l := new(L)
			r1 := l.ReadTree("foo", "bar")
			r2 := l.WriteNode("foo", "baz")
			r2()
			r1()
		})
	})

	t.Run("child locked", func(t *testing.T) {
		testRun(t, "write and read tree", func(t *testing.T) {
			l := new(L)
			r := l.WriteNode("foo", "bar", "baz")
			testLocked(t, l, r, l.ReadTree, "foo", "bar")
		})

		testRun(t, "read tree and write", func(t *testing.T) {
			l := new(L)
			r1 := l.ReadTree("foo", "bar", "baz")
			r2 := l.WriteNode("foo", "bar")
			r2()
			r1()
		})
	})
}

func TestLockWriteTree(t *testing.T) {
	t.Run("root", func(t *testing.T) {
		testRun(t, "unlocked", func(t *testing.T) {
			l := new(L)
			r := l.WriteTree()
			r()
		})

		testRun(t, "locked", func(t *testing.T) {
			l := new(L)
			r := l.WriteTree()
			testLocked(t, l, r, l.WriteTree)
		})
	})

	t.Run("non-root", func(t *testing.T) {
		testRun(t, "unlocked", func(t *testing.T) {
			l := new(L)
			r := l.WriteTree("foo", "bar")
			r()
		})

		testRun(t, "locked", func(t *testing.T) {
			l := new(L)
			r := l.WriteTree("foo", "bar")
			testLocked(t, l, r, l.WriteTree, "foo", "bar")
		})
	})

	t.Run("parent locked", func(t *testing.T) {
		testRun(t, "root", func(t *testing.T) {
			l := new(L)
			r := l.WriteTree()
			testLocked(t, l, r, l.WriteTree, "foo")
		})

		testRun(t, "non-root", func(t *testing.T) {
			l := new(L)
			r := l.WriteTree("foo", "bar")
			testLocked(t, l, r, l.WriteTree, "foo", "bar", "baz")
		})
	})

	testRun(t, "sibling locked", func(t *testing.T) {
		l := new(L)
		r1 := l.WriteTree("foo", "bar")
		r2 := l.WriteTree("foo", "baz")
		r2()
		r1()
	})

	t.Run("child locked", func(t *testing.T) {
		testRun(t, "root", func(t *testing.T) {
			l := new(L)
			r := l.WriteTree("foo")
			testLocked(t, l, r, l.WriteTree, "foo")
		})

		testRun(t, "non-root", func(t *testing.T) {
			l := new(L)
			r := l.WriteTree("foo", "bar", "baz")
			testLocked(t, l, r, l.WriteTree, "foo", "bar")
		})
	})
}

func TestLockReadWriteTree(t *testing.T) {
	t.Run("root", func(t *testing.T) {
		testRun(t, "read and write tree", func(t *testing.T) {
			l := new(L)
			r := l.ReadNode()
			testLocked(t, l, r, l.WriteTree)
		})

		testRun(t, "write tree and read", func(t *testing.T) {
			l := new(L)
			r := l.WriteTree()
			testLocked(t, l, r, l.ReadNode)
		})
	})

	t.Run("non-root", func(t *testing.T) {
		testRun(t, "read and write tree", func(t *testing.T) {
			l := new(L)
			r := l.ReadNode("foo", "bar")
			testLocked(t, l, r, l.WriteTree, "foo", "bar")
		})

		testRun(t, "write tree and read", func(t *testing.T) {
			l := new(L)
			r := l.WriteTree("foo", "bar")
			testLocked(t, l, r, l.ReadNode, "foo", "bar")
		})
	})

	t.Run("parent locked", func(t *testing.T) {
		testRun(t, "read and write tree", func(t *testing.T) {
			l := new(L)
			r1 := l.ReadNode("foo", "bar")
			r2 := l.WriteTree("foo", "bar", "baz")
			r2()
			r1()
		})

		testRun(t, "write tree and read", func(t *testing.T) {
			l := new(L)
			r := l.WriteTree("foo", "bar")
			testLocked(t, l, r, l.ReadNode, "foo", "bar", "baz")
		})
	})

	t.Run("sibling locked", func(t *testing.T) {
		testRun(t, "read and write tree", func(t *testing.T) {
			l := new(L)
			r1 := l.ReadNode("foo", "bar")
			r2 := l.WriteTree("foo", "baz")
			r2()
			r1()
		})

		testRun(t, "write tree and read", func(t *testing.T) {
			l := new(L)
			r1 := l.WriteTree("foo", "bar")
			r2 := l.ReadNode("foo", "baz")
			r2()
			r1()
		})
	})

	t.Run("child locked", func(t *testing.T) {
		testRun(t, "read and write tree", func(t *testing.T) {
			l := new(L)
			r := l.ReadNode("foo", "bar", "baz")
			testLocked(t, l, r, l.WriteTree, "foo", "bar")
		})

		testRun(t, "write tree and read", func(t *testing.T) {
			l := new(L)
			r1 := l.WriteTree("foo", "bar", "baz")
			r2 := l.ReadNode("foo", "bar")
			r2()
			r1()
		})
	})
}

func TestLockWriteWriteTree(t *testing.T) {
	t.Run("root", func(t *testing.T) {
		testRun(t, "write and write tree", func(t *testing.T) {
			l := new(L)
			r := l.WriteNode()
			testLocked(t, l, r, l.WriteTree)
		})

		testRun(t, "write tree and write", func(t *testing.T) {
			l := new(L)
			r := l.WriteTree()
			testLocked(t, l, r, l.WriteNode)
		})
	})

	t.Run("non-root", func(t *testing.T) {
		testRun(t, "write and write tree", func(t *testing.T) {
			l := new(L)
			r := l.WriteNode("foo", "bar")
			testLocked(t, l, r, l.WriteTree, "foo", "bar")
		})

		testRun(t, "write tree and write", func(t *testing.T) {
			l := new(L)
			r := l.WriteTree("foo", "bar")
			testLocked(t, l, r, l.WriteNode, "foo", "bar")
		})
	})

	t.Run("parent locked", func(t *testing.T) {
		testRun(t, "write and write tree", func(t *testing.T) {
			l := new(L)
			r1 := l.WriteNode("foo", "bar")
			r2 := l.WriteTree("foo", "bar", "baz")
			r2()
			r1()
		})

		testRun(t, "write tree and write", func(t *testing.T) {
			l := new(L)
			r := l.WriteTree("foo", "bar")
			testLocked(t, l, r, l.WriteNode, "foo", "bar", "baz")
		})
	})

	t.Run("sibling locked", func(t *testing.T) {
		testRun(t, "write and write tree", func(t *testing.T) {
			l := new(L)
			r1 := l.WriteNode("foo", "bar")
			r2 := l.WriteTree("foo", "baz")
			r2()
			r1()
		})

		testRun(t, "write tree and write", func(t *testing.T) {
			l := new(L)
			r1 := l.WriteTree("foo", "bar")
			r2 := l.WriteNode("foo", "baz")
			r2()
			r1()
		})
	})

	t.Run("child locked", func(t *testing.T) {
		testRun(t, "write and write tree", func(t *testing.T) {
			l := new(L)
			r := l.WriteNode("foo", "bar", "baz")
			testLocked(t, l, r, l.WriteTree, "foo", "bar")
		})

		testRun(t, "write tree and write", func(t *testing.T) {
			l := new(L)
			r1 := l.WriteTree("foo", "bar", "baz")
			r2 := l.WriteNode("foo", "bar")
			r2()
			r1()
		})
	})
}

func TestLockReadTreeWriteTree(t *testing.T) {
	t.Run("root", func(t *testing.T) {
		testRun(t, "read tree and write tree", func(t *testing.T) {
			l := new(L)
			r := l.ReadTree()
			testLocked(t, l, r, l.WriteTree)
		})

		testRun(t, "write tree and read tree", func(t *testing.T) {
			l := new(L)
			r := l.WriteTree()
			testLocked(t, l, r, l.ReadTree)
		})
	})

	t.Run("non-root", func(t *testing.T) {
		testRun(t, "read tree and write tree", func(t *testing.T) {
			l := new(L)
			r := l.ReadTree("foo", "bar")
			testLocked(t, l, r, l.WriteTree, "foo", "bar")
		})

		testRun(t, "write tree and read tree", func(t *testing.T) {
			l := new(L)
			r := l.WriteTree("foo", "bar")
			testLocked(t, l, r, l.ReadTree, "foo", "bar")
		})
	})

	t.Run("parent locked", func(t *testing.T) {
		testRun(t, "read tree and write tree", func(t *testing.T) {
			l := new(L)
			r := l.ReadTree("foo", "bar")
			testLocked(t, l, r, l.WriteTree, "foo", "bar", "baz")
		})

		testRun(t, "write tree and read tree", func(t *testing.T) {
			l := new(L)
			r := l.WriteTree("foo", "bar")
			testLocked(t, l, r, l.ReadTree, "foo", "bar", "baz")
		})
	})

	t.Run("sibling locked", func(t *testing.T) {
		testRun(t, "read tree and write tree", func(t *testing.T) {
			l := new(L)
			r1 := l.ReadTree("foo", "bar")
			r2 := l.WriteTree("foo", "baz")
			r2()
			r1()
		})

		testRun(t, "write tree and read tree", func(t *testing.T) {
			l := new(L)
			r1 := l.WriteTree("foo", "bar")
			r2 := l.ReadTree("foo", "baz")
			r2()
			r1()
		})
	})

	t.Run("child locked", func(t *testing.T) {
		testRun(t, "read tree and write tree", func(t *testing.T) {
			l := new(L)
			r := l.ReadTree("foo", "bar", "baz")
			testLocked(t, l, r, l.WriteTree, "foo", "bar")
		})

		testRun(t, "write tree and read tree", func(t *testing.T) {
			l := new(L)
			r := l.WriteTree("foo", "bar", "baz")
			testLocked(t, l, r, l.ReadTree, "foo", "bar")
		})
	})
}

func TestLockRace(t *testing.T) {
	l := new(L)

	access := func() <-chan struct{} {
		done := make(chan struct{})
		go func() {
			r := l.WriteNode()
			r()
			close(done)
		}()

		return done
	}

	done1 := access()
	done2 := access()
	<-done1
	<-done2
}
