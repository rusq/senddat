package senddat

import (
	"reflect"
	"testing"
)

var sampleTrieComspecs = []CommandSpec{
	{
		// 0
		Prefix:   []byte{0x1B, '@'},
		Name:     "Initialize Printer",
		ArgCount: 0,
	},
	{
		// 1
		Prefix:   []byte{0x1B, 'i'},
		Name:     "Select Character Set",
		ArgCount: 1,
	},
	{
		// 2
		Prefix:   []byte{0x1B, '(', 'A'},
		Name:     "Select Character Set 1",
		ArgCount: 0,
	},
	{
		// 3
		Prefix:   []byte{byte(bGS), '(', 'k'},
		Name:     "Set Barcode Height",
		ArgCount: 1,
	},
	{
		// 4
		Prefix:   []byte{byte(bGS), '(', 'V'},
		Name:     "Paper cut",
		ArgCount: 1,
	},
	{
		// 5
		Prefix: []byte{byte(bHT)},
		Name:   "Horizontal Tab",
	},
	{
		// 6
		Prefix: []byte{byte(bLF)},
		Name:   "Print and Line Feed",
	},
	{
		// 7
		Prefix: []byte{0x1B, '(', 'Y'},
		Name:   "Specify Batch Print",
	},
}

func Test_buildTrie(t *testing.T) {
	type args struct {
		cmds []CommandSpec
	}
	tests := []struct {
		name string
		args args
		want *trieNode
	}{
		{
			name: "sampleTrieComspecs",
			args: args{
				cmds: sampleTrieComspecs,
			},
			want: &trieNode{
				children: map[byte]*trieNode{
					0x1B: {
						children: map[byte]*trieNode{
							0x40: {
								children: make(map[byte]*trieNode),
								spec:     &sampleTrieComspecs[0],
							},
							'i': {
								children: map[byte]*trieNode{},
								spec:     &sampleTrieComspecs[1],
							},
							'(': {
								children: map[byte]*trieNode{
									'A': {
										children: make(map[byte]*trieNode),
										spec:     &sampleTrieComspecs[2],
									},
									'Y': {
										children: make(map[byte]*trieNode),
										spec:     &sampleTrieComspecs[7],
									},
								},
							},
						},
						spec: nil,
					},
					byte(bGS): {
						children: map[byte]*trieNode{
							'(': {
								children: map[byte]*trieNode{
									'k': {
										children: make(map[byte]*trieNode),
										spec:     &sampleTrieComspecs[3],
									},
									'V': {
										children: make(map[byte]*trieNode),
										spec:     &sampleTrieComspecs[4],
									},
								},
								spec: nil, // This node is not an end node
							},
						},
					},
					byte(bHT): {
						children: make(map[byte]*trieNode),
						spec:     &sampleTrieComspecs[5],
					},
					byte(bLF): {
						children: make(map[byte]*trieNode),
						spec:     &sampleTrieComspecs[6],
					},
				},
				spec: nil, // Root node does not have a command spec
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildTrie(tt.args.cmds); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("buildTrie() = %v, want %v", got, tt.want)
			}
		})
	}
}
