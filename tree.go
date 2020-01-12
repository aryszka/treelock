package treelock

type node struct {
	operations        listRange
	subtreeOperations listRange
	children          map[string]*node
}

func nodePath(from *node, path []string) []*node {
	np := make([]*node, 1, len(path)+1)
	np[0] = from
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

func insert(nodePath []*node, o *operation) {
	o.item = &item{operation: o}
	n, nodePath := nodePath[len(nodePath)-1], nodePath[:len(nodePath)-1]
	n.operations = insertTo(n.operations, o.item)
	connect(n.operations, n.subtreeOperations)
	for j := len(nodePath) - 1; j >= 0; j-- {
		n = nodePath[j]
		n.subtreeOperations = insertTo(n.subtreeOperations, o.item)
		connect(n.operations, n.subtreeOperations)
	}
}

func remove(nodePath []*node, o *operation) {
	n := nodePath[len(nodePath)-1]
	n.operations = removeFrom(n.operations, o.item)
	for j := len(nodePath) - 1; j >= 0; j-- {
		n = nodePath[j]
		n.subtreeOperations = removeFrom(n.subtreeOperations, o.item)
		if j > 0 && n.operations.empty() && n.subtreeOperations.empty() {
			delete(nodePath[j-1].children, o.path[j-1])
		}
	}
}
