package treelock

type node struct {
	items        list
	subtreeItems list
	children     map[string]*node
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

func (t *tree) insert(nodePath []*node, i *item) {
	i.element = &element{item: i}
	n, nodePath := nodePath[len(nodePath)-1], nodePath[:len(nodePath)-1]
	n.items = n.items.insert(i.element)
	connect(n.items, n.subtreeItems)
	for j := len(nodePath) - 1; j >= 0; j-- {
		n = nodePath[j]
		n.subtreeItems = n.subtreeItems.insert(i.element)
		connect(n.items, n.subtreeItems)
	}
}

func (t *tree) remove(i *item) {
	np := t.nodePath(i.path)
	n := np[len(np)-1]
	n.items = n.items.remove(i.element)
	for j := len(np) - 1; j >= 0; j-- {
		n = np[j]
		n.subtreeItems = n.subtreeItems.remove(i.element)
		if j > 0 && n.items.empty() && n.subtreeItems.empty() {
			delete(np[j-1].children, i.path[j-1])
		}
	}
}
