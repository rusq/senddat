package senddat

type trieNode struct {
	children map[byte]*trieNode
	spec     *CommandSpec // Pointer to the command spec if this node is an end node
}

func buildTrie(specs []CommandSpec) *trieNode {
	root := &trieNode{children: make(map[byte]*trieNode)}

	for _, spec := range specs {
		current := root
		for _, b := range spec.Prefix {
			if _, exists := current.children[b]; !exists {
				current.children[b] = &trieNode{children: make(map[byte]*trieNode)}
			}
			current = current.children[b]
		}
		current.spec = &spec // Set the command spec at the end of the prefix
	}

	return root
}

func (n *trieNode) findChild(b byte) (*trieNode, bool) {
	child, exists := n.children[b]
	return child, exists
}
