package senddat

import (
	"io"
)

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

func (n *trieNode) findCommand(r io.ByteReader) (*CommandSpec, bool, error) {
	current := n
	for {
		b, err := r.ReadByte()
		if err != nil {
			if err == io.EOF {
				return nil, false, io.ErrUnexpectedEOF // No command found
			}
			return nil, false, err // Error reading byte
		}

		if nextNode, exists := current.children[b]; exists {
			current = nextNode
			if current.spec != nil {
				return current.spec, true, nil // Found a command spec
			}
		} else {
			return nil, false, nil // No matching command found
		}
	}
}
