package senddat

import (
	"bytes"
	"io"
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

func Test_trieNode_findCommand(t *testing.T) {
	type args struct {
		r io.ByteReader
	}
	tests := []struct {
		name    string
		nodes   *trieNode
		args    args
		want    *CommandSpec
		want1   bool
		wantErr bool
	}{
		{
			name:  "HT",
			nodes: buildTrie(sampleTrieComspecs),
			args: args{
				r: bytes.NewReader([]byte{byte(bHT)}),
			},
			want:    &sampleTrieComspecs[5],
			want1:   true,
			wantErr: false,
		},
		{
			name:  "LF",
			nodes: buildTrie(sampleTrieComspecs),
			args: args{
				r: bytes.NewReader([]byte{byte(bLF)}),
			},
			want:    &sampleTrieComspecs[6],
			want1:   true,
			wantErr: false,
		},
		{
			name:  "ESC @",
			nodes: buildTrie(sampleTrieComspecs),
			args: args{
				r: bytes.NewReader([]byte{0x1B, 0x40}),
			},
			want:    &sampleTrieComspecs[0],
			want1:   true,
			wantErr: false,
		},
		{
			name:  "ESC o", // not found
			nodes: buildTrie(sampleTrieComspecs),
			args: args{
				r: bytes.NewReader([]byte{0x1B, 'o'}),
			},
			want:    nil,
			want1:   false,
			wantErr: false,
		},
		{
			name:  "ESC i 1",
			nodes: buildTrie(sampleTrieComspecs),
			args: args{
				r: bytes.NewReader([]byte{0x1B, 0x69}),
			},
			want:    &sampleTrieComspecs[1],
			want1:   true,
			wantErr: false,
		},
		{
			name:  "ESC (incomplete command)",
			nodes: buildTrie(sampleTrieComspecs),
			args: args{
				r: bytes.NewReader([]byte{0x1B}),
			},
			want:    nil,
			want1:   false,
			wantErr: true,
		},
		{
			name:  "ESC ( A",
			nodes: buildTrie(sampleTrieComspecs),
			args: args{
				r: bytes.NewReader([]byte{0x1B, '(', 'A'}),
			},
			want:    &sampleTrieComspecs[2],
			want1:   true,
			wantErr: false,
		},
		{
			name:  "GS ( k",
			nodes: buildTrie(sampleTrieComspecs),
			args: args{
				r: bytes.NewReader([]byte{byte(bGS), '(', 'k'}),
			},
			want:    &sampleTrieComspecs[3],
			want1:   true,
			wantErr: false,
		},
		{
			name:  "ESC ( Y",
			nodes: buildTrie(sampleTrieComspecs),
			args: args{
				r: bytes.NewReader([]byte{0x1B, '(', 'Y'}),
			},
			want:    &sampleTrieComspecs[7],
			want1:   true,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := tt.nodes.findCommand(tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("trieNode.findCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("trieNode.findCommand() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("trieNode.findCommand() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
