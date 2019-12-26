package treelock

import (
	"testing"
	"time"
)

const minDelay = 9 * time.Millisecond

func testLocked(t *testing.T, l *Lock, release releaseLock, method func(...string) releaseLock, path ...string) {
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

func TestLockRead(t *testing.T) {
	t.Run("root", func(t *testing.T) {
		t.Run("single", func(t *testing.T) {
			l := New()
			defer l.Close()
			r := l.ReadNode()
			r()
		})

		t.Run("multiple", func(t *testing.T) {
			l := New()
			defer l.Close()
			r1 := l.ReadNode()
			r2 := l.ReadNode()
			r1()
			r2()
		})
	})

	t.Run("non-root", func(t *testing.T) {
		t.Run("single", func(t *testing.T) {
			l := New()
			defer l.Close()
			r := l.ReadNode("foo", "bar")
			r()
		})

		t.Run("multiple", func(t *testing.T) {
			l := New()
			defer l.Close()
			r1 := l.ReadNode("foo", "bar")
			r2 := l.ReadNode("foo", "bar")
			r1()
			r2()
		})
	})

	t.Run("parent locked", func(t *testing.T) {
		l := New()
		defer l.Close()
		r1 := l.ReadNode("foo")
		r2 := l.ReadNode("foo", "bar")
		r2()
		r1()
	})

	t.Run("sibling locked", func(t *testing.T) {
		l := New()
		defer l.Close()
		r1 := l.ReadNode("foo", "bar")
		r2 := l.ReadNode("foo", "baz")
		r2()
		r1()
	})

	t.Run("child locked", func(t *testing.T) {
		l := New()
		defer l.Close()
		r1 := l.ReadNode("foo", "bar", "baz")
		r2 := l.ReadNode("foo", "bar")
		r2()
		r1()
	})
}

func TestLockWrite(t *testing.T) {
	t.Run("root", func(t *testing.T) {
		t.Run("unlocked", func(t *testing.T) {
			l := New()
			defer l.Close()
			r := l.WriteNode()
			r()
		})

		t.Run("locked", func(t *testing.T) {
			l := New()
			defer l.Close()
			r := l.WriteNode()
			testLocked(t, l, r, l.WriteNode)
		})
	})

	t.Run("non-root", func(t *testing.T) {
		t.Run("unlocked", func(t *testing.T) {
			l := New()
			defer l.Close()
			r := l.WriteNode("foo", "bar")
			r()
		})

		t.Run("locked", func(t *testing.T) {
			l := New()
			defer l.Close()
			r := l.WriteNode("foo", "bar")
			testLocked(t, l, r, l.WriteNode, "foo", "bar")
		})
	})

	t.Run("parent locked", func(t *testing.T) {
		l := New()
		defer l.Close()
		r1 := l.WriteNode("foo")
		r2 := l.WriteNode("foo", "bar")
		r2()
		r1()
	})

	t.Run("sibling locked", func(t *testing.T) {
		l := New()
		defer l.Close()
		r1 := l.WriteNode("foo", "bar")
		r2 := l.WriteNode("foo", "baz")
		r2()
		r1()
	})

	t.Run("child locked", func(t *testing.T) {
		l := New()
		defer l.Close()
		r1 := l.WriteNode("foo", "bar", "baz")
		r2 := l.WriteNode("foo", "bar")
		r2()
		r1()
	})
}

func TestLockReadWrite(t *testing.T) {
	t.Run("root", func(t *testing.T) {
		t.Run("read and write", func(t *testing.T) {
			l := New()
			defer l.Close()
			r := l.ReadNode()
			testLocked(t, l, r, l.WriteNode)
		})

		t.Run("write and read", func(t *testing.T) {
			l := New()
			defer l.Close()
			r := l.WriteNode()
			testLocked(t, l, r, l.ReadNode)
		})
	})

	t.Run("non-root", func(t *testing.T) {
		t.Run("read and write", func(t *testing.T) {
			l := New()
			defer l.Close()
			r := l.ReadNode("foo", "bar")
			testLocked(t, l, r, l.WriteNode, "foo", "bar")
		})

		t.Run("write and read", func(t *testing.T) {
			l := New()
			defer l.Close()
			r := l.WriteNode("foo", "bar")
			testLocked(t, l, r, l.ReadNode, "foo", "bar")
		})
	})

	t.Run("parent locked", func(t *testing.T) {
		t.Run("read and write", func(t *testing.T) {
			l := New()
			defer l.Close()
			r1 := l.ReadNode("foo")
			r2 := l.WriteNode("foo", "bar")
			r2()
			r1()
		})

		t.Run("write and read", func(t *testing.T) {
			l := New()
			defer l.Close()
			r1 := l.WriteNode("foo")
			r2 := l.ReadNode("foo", "bar")
			r2()
			r1()
		})
	})

	t.Run("sibling locked", func(t *testing.T) {
		t.Run("read and write", func(t *testing.T) {
			l := New()
			defer l.Close()
			r1 := l.ReadNode("foo", "bar")
			r2 := l.WriteNode("foo", "baz")
			r2()
			r1()
		})

		t.Run("write and read", func(t *testing.T) {
			l := New()
			defer l.Close()
			r1 := l.WriteNode("foo", "bar")
			r2 := l.ReadNode("foo", "baz")
			r2()
			r1()
		})
	})

	t.Run("child locked", func(t *testing.T) {
		t.Run("read and write", func(t *testing.T) {
			l := New()
			defer l.Close()
			r1 := l.ReadNode("foo", "bar", "baz")
			r2 := l.WriteNode("foo", "bar")
			r2()
			r1()
		})

		t.Run("write and read", func(t *testing.T) {
			l := New()
			defer l.Close()
			r1 := l.WriteNode("foo", "bar", "baz")
			r2 := l.ReadNode("foo", "bar")
			r2()
			r1()
		})
	})
}

func TestLockReadTree(t *testing.T) {
	t.Run("root", func(t *testing.T) {
		t.Run("single", func(t *testing.T) {
			l := New()
			defer l.Close()
			r := l.ReadTree()
			r()
		})

		t.Run("multiple", func(t *testing.T) {
			l := New()
			defer l.Close()
			r1 := l.ReadTree()
			r2 := l.ReadTree()
			r2()
			r1()
		})
	})

	t.Run("non-root", func(t *testing.T) {
		t.Run("single", func(t *testing.T) {
			l := New()
			defer l.Close()
			r := l.ReadTree("foo", "bar")
			r()
		})

		t.Run("multiple", func(t *testing.T) {
			l := New()
			defer l.Close()
			r1 := l.ReadTree("foo", "bar")
			r2 := l.ReadTree("foo", "bar")
			r2()
			r1()
		})
	})

	t.Run("parent locked", func(t *testing.T) {
		t.Run("root", func(t *testing.T) {
			l := New()
			defer l.Close()
			r1 := l.ReadTree()
			r2 := l.ReadTree("foo", "bar")
			r2()
			r1()
		})

		t.Run("non-root", func(t *testing.T) {
			l := New()
			defer l.Close()
			r1 := l.ReadTree("foo", "bar")
			r2 := l.ReadTree("foo", "bar", "baz")
			r2()
			r1()
		})
	})

	t.Run("sibling locked", func(t *testing.T) {
		l := New()
		defer l.Close()
		r1 := l.ReadTree("foo", "bar")
		r2 := l.ReadTree("foo", "baz")
		r2()
		r1()
	})

	t.Run("child locked", func(t *testing.T) {
		t.Run("root", func(t *testing.T) {
			l := New()
			defer l.Close()
			r1 := l.ReadTree("foo", "bar")
			r2 := l.ReadTree()
			r2()
			r1()
		})

		t.Run("non-root", func(t *testing.T) {
			l := New()
			defer l.Close()
			r1 := l.ReadTree("foo", "bar", "baz")
			r2 := l.ReadTree("foo", "bar")
			r2()
			r1()
		})
	})
}

func TestLockReadReadTree(t *testing.T) {
	t.Run("root", func(t *testing.T) {
		t.Run("read and read tree", func(t *testing.T) {
			l := New()
			defer l.Close()
			r1 := l.ReadNode()
			r2 := l.ReadTree()
			r2()
			r1()
		})

		t.Run("read tree and read", func(t *testing.T) {
			l := New()
			defer l.Close()
			r1 := l.ReadTree()
			r2 := l.ReadNode()
			r2()
			r1()
		})
	})

	t.Run("non-root", func(t *testing.T) {
		t.Run("read and read tree", func(t *testing.T) {
			l := New()
			defer l.Close()
			r1 := l.ReadNode("foo", "bar")
			r2 := l.ReadTree("foo", "bar")
			r2()
			r1()
		})

		t.Run("read tree and read", func(t *testing.T) {
			l := New()
			defer l.Close()
			r1 := l.ReadTree("foo", "bar")
			r2 := l.ReadNode("foo", "bar")
			r2()
			r1()
		})
	})

	t.Run("parent locked", func(t *testing.T) {
		t.Run("read and read tree", func(t *testing.T) {
			l := New()
			defer l.Close()
			r1 := l.ReadNode("foo", "bar")
			r2 := l.ReadTree("foo", "bar", "baz")
			r2()
			r1()
		})

		t.Run("read tree and read", func(t *testing.T) {
			l := New()
			defer l.Close()
			r1 := l.ReadTree("foo", "bar")
			r2 := l.ReadNode("foo", "bar", "baz")
			r2()
			r1()
		})
	})

	t.Run("sibling locked", func(t *testing.T) {
		t.Run("read and read tree", func(t *testing.T) {
			l := New()
			defer l.Close()
			r1 := l.ReadNode("foo", "bar")
			r2 := l.ReadTree("foo", "baz")
			r2()
			r1()
		})

		t.Run("read tree and read", func(t *testing.T) {
			l := New()
			defer l.Close()
			r1 := l.ReadTree("foo", "bar")
			r2 := l.ReadNode("foo", "baz")
			r2()
			r1()
		})
	})

	t.Run("child locked", func(t *testing.T) {
		t.Run("read and read tree", func(t *testing.T) {
			l := New()
			defer l.Close()
			r1 := l.ReadNode("foo", "bar", "baz")
			r2 := l.ReadTree("foo", "bar")
			r2()
			r1()
		})

		t.Run("read tree and read", func(t *testing.T) {
			l := New()
			defer l.Close()
			r1 := l.ReadTree("foo", "bar", "baz")
			r2 := l.ReadNode("foo", "bar")
			r2()
			r1()
		})
	})
}

func TestLockWriteReadTree(t *testing.T) {
	t.Run("root", func(t *testing.T) {
		t.Run("write and read tree", func(t *testing.T) {
			l := New()
			defer l.Close()
			r := l.WriteNode()
			testLocked(t, l, r, l.ReadTree)
		})

		t.Run("read tree and write", func(t *testing.T) {
			l := New()
			defer l.Close()
			r := l.ReadTree()
			testLocked(t, l, r, l.WriteNode)
		})
	})

	t.Run("non-root", func(t *testing.T) {
		t.Run("write and read tree", func(t *testing.T) {
			l := New()
			defer l.Close()
			r := l.WriteNode("foo", "bar")
			testLocked(t, l, r, l.ReadTree, "foo", "bar")
		})

		t.Run("read tree and write", func(t *testing.T) {
			l := New()
			defer l.Close()
			r := l.ReadTree("foo", "bar")
			testLocked(t, l, r, l.WriteNode, "foo", "bar")
		})
	})

	t.Run("parent locked", func(t *testing.T) {
		t.Run("write and read tree", func(t *testing.T) {
			l := New()
			defer l.Close()
			r1 := l.WriteNode("foo", "bar")
			r2 := l.ReadTree("foo", "bar", "baz")
			r2()
			r1()
		})

		t.Run("read tree and write", func(t *testing.T) {
			l := New()
			defer l.Close()
			r := l.ReadTree("foo", "bar")
			testLocked(t, l, r, l.WriteNode, "foo", "bar", "baz")
		})
	})

	t.Run("sibling locked", func(t *testing.T) {
		t.Run("write and read tree", func(t *testing.T) {
			l := New()
			defer l.Close()
			r1 := l.WriteNode("foo", "bar")
			r2 := l.ReadTree("foo", "baz")
			r2()
			r1()
		})

		t.Run("read tree and write", func(t *testing.T) {
			l := New()
			defer l.Close()
			r1 := l.ReadTree("foo", "bar")
			r2 := l.WriteNode("foo", "baz")
			r2()
			r1()
		})
	})

	t.Run("child locked", func(t *testing.T) {
		t.Run("write and read tree", func(t *testing.T) {
			l := New()
			defer l.Close()
			r := l.WriteNode("foo", "bar", "baz")
			testLocked(t, l, r, l.ReadTree, "foo", "bar")
		})

		t.Run("read tree and write", func(t *testing.T) {
			l := New()
			defer l.Close()
			r1 := l.ReadTree("foo", "bar", "baz")
			r2 := l.WriteNode("foo", "bar")
			r2()
			r1()
		})
	})
}

func TestLockWriteTree(t *testing.T) {
	t.Run("root", func(t *testing.T) {
		t.Run("unlocked", func(t *testing.T) {
			l := New()
			defer l.Close()
			r := l.WriteTree()
			r()
		})

		t.Run("locked", func(t *testing.T) {
			l := New()
			defer l.Close()
			r := l.WriteTree()
			testLocked(t, l, r, l.WriteTree)
		})
	})

	t.Run("non-root", func(t *testing.T) {
		t.Run("unlocked", func(t *testing.T) {
			l := New()
			defer l.Close()
			r := l.WriteTree("foo", "bar")
			r()
		})

		t.Run("locked", func(t *testing.T) {
			l := New()
			defer l.Close()
			r := l.WriteTree("foo", "bar")
			testLocked(t, l, r, l.WriteTree, "foo", "bar")
		})
	})

	t.Run("parent locked", func(t *testing.T) {
		t.Run("root", func(t *testing.T) {
			l := New()
			defer l.Close()
			r := l.WriteTree()
			testLocked(t, l, r, l.WriteTree, "foo")
		})

		t.Run("non-root", func(t *testing.T) {
			l := New()
			defer l.Close()
			r := l.WriteTree("foo", "bar")
			testLocked(t, l, r, l.WriteTree, "foo", "bar", "baz")
		})
	})

	t.Run("sibling locked", func(t *testing.T) {
		l := New()
		defer l.Close()
		r1 := l.WriteTree("foo", "bar")
		r2 := l.WriteTree("foo", "baz")
		r2()
		r1()
	})

	t.Run("child locked", func(t *testing.T) {
		t.Run("root", func(t *testing.T) {
			l := New()
			defer l.Close()
			r := l.WriteTree("foo")
			testLocked(t, l, r, l.WriteTree, "foo")
		})

		t.Run("non-root", func(t *testing.T) {
			l := New()
			defer l.Close()
			r := l.WriteTree("foo", "bar", "baz")
			testLocked(t, l, r, l.WriteTree, "foo", "bar")
		})
	})
}

func TestLockReadWriteTree(t *testing.T) {
	t.Run("root", func(t *testing.T) {
		t.Run("read and write tree", func(t *testing.T) {
			l := New()
			defer l.Close()
			r := l.ReadNode()
			testLocked(t, l, r, l.WriteTree)
		})

		t.Run("write tree and read", func(t *testing.T) {
			l := New()
			defer l.Close()
			r := l.WriteTree()
			testLocked(t, l, r, l.ReadNode)
		})
	})

	t.Run("non-root", func(t *testing.T) {
		t.Run("read and write tree", func(t *testing.T) {
			l := New()
			defer l.Close()
			r := l.ReadNode("foo", "bar")
			testLocked(t, l, r, l.WriteTree, "foo", "bar")
		})

		t.Run("write tree and read", func(t *testing.T) {
			l := New()
			defer l.Close()
			r := l.WriteTree("foo", "bar")
			testLocked(t, l, r, l.ReadNode, "foo", "bar")
		})
	})

	t.Run("parent locked", func(t *testing.T) {
		t.Run("read and write tree", func(t *testing.T) {
			l := New()
			defer l.Close()
			r1 := l.ReadNode("foo", "bar")
			r2 := l.WriteTree("foo", "bar", "baz")
			r2()
			r1()
		})

		t.Run("write tree and read", func(t *testing.T) {
			l := New()
			defer l.Close()
			r := l.WriteTree("foo", "bar")
			testLocked(t, l, r, l.ReadNode, "foo", "bar", "baz")
		})
	})

	t.Run("sibling locked", func(t *testing.T) {
		t.Run("read and write tree", func(t *testing.T) {
			l := New()
			defer l.Close()
			r1 := l.ReadNode("foo", "bar")
			r2 := l.WriteTree("foo", "baz")
			r2()
			r1()
		})

		t.Run("write tree and read", func(t *testing.T) {
			l := New()
			defer l.Close()
			r1 := l.WriteTree("foo", "bar")
			r2 := l.ReadNode("foo", "baz")
			r2()
			r1()
		})
	})

	t.Run("child locked", func(t *testing.T) {
		t.Run("read and write tree", func(t *testing.T) {
			l := New()
			defer l.Close()
			r := l.ReadNode("foo", "bar", "baz")
			testLocked(t, l, r, l.WriteTree, "foo", "bar")
		})

		t.Run("write tree and read", func(t *testing.T) {
			l := New()
			defer l.Close()
			r1 := l.WriteTree("foo", "bar", "baz")
			r2 := l.ReadNode("foo", "bar")
			r2()
			r1()
		})
	})
}

func TestLockWriteWriteTree(t *testing.T) {
	t.Run("root", func(t *testing.T) {
		t.Run("write and write tree", func(t *testing.T) {
			l := New()
			defer l.Close()
			r := l.WriteNode()
			testLocked(t, l, r, l.WriteTree)
		})

		t.Run("write tree and write", func(t *testing.T) {
			l := New()
			defer l.Close()
			r := l.WriteTree()
			testLocked(t, l, r, l.WriteNode)
		})
	})

	t.Run("non-root", func(t *testing.T) {
		t.Run("write and write tree", func(t *testing.T) {
			l := New()
			defer l.Close()
			r := l.WriteNode("foo", "bar")
			testLocked(t, l, r, l.WriteTree, "foo", "bar")
		})

		t.Run("write tree and write", func(t *testing.T) {
			l := New()
			defer l.Close()
			r := l.WriteTree("foo", "bar")
			testLocked(t, l, r, l.WriteNode, "foo", "bar")
		})
	})

	t.Run("parent locked", func(t *testing.T) {
		t.Run("write and write tree", func(t *testing.T) {
			l := New()
			defer l.Close()
			r1 := l.WriteNode("foo", "bar")
			r2 := l.WriteTree("foo", "bar", "baz")
			r2()
			r1()
		})

		t.Run("write tree and write", func(t *testing.T) {
			l := New()
			defer l.Close()
			r := l.WriteTree("foo", "bar")
			testLocked(t, l, r, l.WriteNode, "foo", "bar", "baz")
		})
	})

	t.Run("sibling locked", func(t *testing.T) {
		t.Run("write and write tree", func(t *testing.T) {
			l := New()
			defer l.Close()
			r1 := l.WriteNode("foo", "bar")
			r2 := l.WriteTree("foo", "baz")
			r2()
			r1()
		})

		t.Run("write tree and write", func(t *testing.T) {
			l := New()
			defer l.Close()
			r1 := l.WriteTree("foo", "bar")
			r2 := l.WriteNode("foo", "baz")
			r2()
			r1()
		})
	})

	t.Run("child locked", func(t *testing.T) {
		t.Run("write and write tree", func(t *testing.T) {
			l := New()
			defer l.Close()
			r := l.WriteNode("foo", "bar", "baz")
			testLocked(t, l, r, l.WriteTree, "foo", "bar")
		})

		t.Run("write tree and write", func(t *testing.T) {
			l := New()
			defer l.Close()
			r1 := l.WriteTree("foo", "bar", "baz")
			r2 := l.WriteNode("foo", "bar")
			r2()
			r1()
		})
	})
}

func TestLockReadTreeWriteTree(t *testing.T) {
	t.Run("root", func(t *testing.T) {
		t.Run("read tree and write tree", func(t *testing.T) {
			l := New()
			defer l.Close()
			r := l.ReadTree()
			testLocked(t, l, r, l.WriteTree)
		})

		t.Run("write tree and read tree", func(t *testing.T) {
			l := New()
			defer l.Close()
			r := l.WriteTree()
			testLocked(t, l, r, l.ReadTree)
		})
	})

	t.Run("non-root", func(t *testing.T) {
		t.Run("read tree and write tree", func(t *testing.T) {
			l := New()
			defer l.Close()
			r := l.ReadTree("foo", "bar")
			testLocked(t, l, r, l.WriteTree, "foo", "bar")
		})

		t.Run("write tree and read tree", func(t *testing.T) {
			l := New()
			defer l.Close()
			r := l.WriteTree("foo", "bar")
			testLocked(t, l, r, l.ReadTree, "foo", "bar")
		})
	})

	t.Run("parent locked", func(t *testing.T) {
		t.Run("read tree and write tree", func(t *testing.T) {
			l := New()
			defer l.Close()
			r := l.ReadTree("foo", "bar")
			testLocked(t, l, r, l.WriteTree, "foo", "bar", "baz")
		})

		t.Run("write tree and read tree", func(t *testing.T) {
			l := New()
			defer l.Close()
			r := l.WriteTree("foo", "bar")
			testLocked(t, l, r, l.ReadTree, "foo", "bar", "baz")
		})
	})

	t.Run("sibling locked", func(t *testing.T) {
		t.Run("read tree and write tree", func(t *testing.T) {
			l := New()
			defer l.Close()
			r1 := l.ReadTree("foo", "bar")
			r2 := l.WriteTree("foo", "baz")
			r2()
			r1()
		})

		t.Run("write tree and read tree", func(t *testing.T) {
			l := New()
			defer l.Close()
			r1 := l.WriteTree("foo", "bar")
			r2 := l.ReadTree("foo", "baz")
			r2()
			r1()
		})
	})

	t.Run("child locked", func(t *testing.T) {
		t.Run("read tree and write tree", func(t *testing.T) {
			l := New()
			defer l.Close()
			r := l.ReadTree("foo", "bar", "baz")
			testLocked(t, l, r, l.WriteTree, "foo", "bar")
		})

		t.Run("write tree and read tree", func(t *testing.T) {
			l := New()
			defer l.Close()
			r := l.WriteTree("foo", "bar", "baz")
			testLocked(t, l, r, l.ReadTree, "foo", "bar")
		})
	})
}

func TestLockRace(t *testing.T) {
	l := New()
	defer l.Close()

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
