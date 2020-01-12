package treelock_test

import (
	"strings"

	"github.com/aryszka/treelock"
)

func writeFile(l *treelock.L, path string) {
	p := strings.Split(path, "/")[1:]
	release := l.WriteNode(p...)
	defer release()
	// write to file with path
}

func Example() {
	l := new(treelock.L)
	writeFile(l, "/a/b/c")
}
