package treelock

type node struct {
	operations        list
	subtreeOperations list
	children          map[string]*node
}

type tree struct {
	root *node
}

func newTree() *tree {
	return &tree{&node{}}
}

func (t *tree) nodePath(path []string) []*node {
	np := []*node{t.root}
	for _, p := range path {
		n, ok := np[len(np)-1].children[p]
		if !ok {
			if np[len(np)-1].children == nil {
				np[len(np)-1].children = make(map[string]*node)
			}

			n = &node{}
			np[len(np)-1].children[p] = n
		}

		np = append(np, n)
	}

	return np
}

func (t *tree) insert(nodePath []*node, o *operation) {
	o.item = &item{operation: o}
	n, nodePath := nodePath[len(nodePath)-1], nodePath[:len(nodePath)-1]
	n.operations = n.operations.insert(o.item)
	connect(n.operations, n.subtreeOperations)
	for j := len(nodePath) - 1; j >= 0; j-- {
		n = nodePath[j]
		n.subtreeOperations = n.subtreeOperations.insert(o.item)
		connect(n.operations, n.subtreeOperations)
	}
}

func (t *tree) remove(o *operation) {
	np := t.nodePath(o.path)
	n := np[len(np)-1]
	n.operations = n.operations.remove(o.item)
	for j := len(np) - 1; j >= 0; j-- {
		n = np[j]
		n.subtreeOperations = n.subtreeOperations.remove(o.item)
		if j > 0 && n.operations.empty() && n.subtreeOperations.empty() {
			delete(np[j-1].children, o.path[j-1])
		}
	}
}
