package senddat

type trieNode struct {
	children map[byte]*trieNode
	spec     *CommandSpec // Pointer to the command spec if this node is an end node
}

func buildTrie(cmds []CommandSpec) *trieNode {
	root := &trieNode{children: make(map[byte]*trieNode)}

	for _, cmd := range cmds {
		current := root
		for _, b := range cmd.Prefix {
			if _, exists := current.children[b]; !exists {
				current.children[b] = &trieNode{children: make(map[byte]*trieNode)}
			}
			current = current.children[b]
		}
		current.spec = &cmd // Set the command spec at the end of the prefix
	}

	return root
}
