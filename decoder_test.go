package senddat

import (
	"bytes"
	_ "embed"
	"io"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

var genericComspecs []CommandSpec

func init() {
	var err error
	genericComspecs, err = loadCommandSpecs("drivers/escpos-3.40.csv")
	if err != nil {
		panic(err)
	}
}

func TestDecode(t *testing.T) {
	type args struct {
		r        io.Reader
		comspecs []CommandSpec
	}
	tests := []struct {
		name    string
		args    args
		want    []Entry
		wantErr bool
	}{
		{
			name: "testESC",
			args: args{
				r:        bytes.NewReader(loadTestFile(t, "testdata/POS/80x72.prn")),
				comspecs: genericComspecs,
			},
			want:    []Entry{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Decode(tt.args.r, tt.args.comspecs)
			if (err != nil) != tt.wantErr {
				t.Errorf("Decode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got, "Decode() = %v, want %v", got, tt.want)
		})
	}
}

func Test_findComSpec(t *testing.T) {
	type args struct {
		nodes *trieNode
		r     io.ByteScanner
	}
	tests := []struct {
		name    string
		args    args
		want    *CommandSpec
		want1   int
		wantErr bool
	}{
		{
			name: "HT",
			args: args{
				nodes: buildTrie(sampleTrieComspecs),
				r:     bytes.NewReader([]byte{byte(bHT)}),
			},
			want:    &sampleTrieComspecs[5],
			want1:   1,
			wantErr: false,
		},
		{
			name: "LF",
			args: args{
				nodes: buildTrie(sampleTrieComspecs),
				r:     bytes.NewReader([]byte{byte(bLF)}),
			},
			want:    &sampleTrieComspecs[6],
			want1:   1,
			wantErr: false,
		},
		{
			name: "ESC @",
			args: args{
				nodes: buildTrie(sampleTrieComspecs),
				r:     bytes.NewReader([]byte{0x1B, 0x40}),
			},
			want:    &sampleTrieComspecs[0],
			want1:   2,
			wantErr: false,
		},
		{
			name: "ESC o", // not found
			args: args{
				nodes: buildTrie(sampleTrieComspecs),
				r:     bytes.NewReader([]byte{0x1B, 'o'}),
			},
			want:    nil,
			want1:   2,
			wantErr: true,
		},
		{
			name: "ESC i 1",
			args: args{
				nodes: buildTrie(sampleTrieComspecs),
				r:     bytes.NewReader([]byte{0x1B, 0x69}),
			},
			want:    &sampleTrieComspecs[1],
			want1:   2,
			wantErr: false,
		},
		{
			name: "ESC (incomplete command)",
			args: args{
				nodes: buildTrie(sampleTrieComspecs),
				r:     bytes.NewReader([]byte{0x1B}),
			},
			want:    nil,
			want1:   1,
			wantErr: true,
		},
		{
			name: "ESC ( A",
			args: args{
				nodes: buildTrie(sampleTrieComspecs),
				r:     bytes.NewReader([]byte{0x1B, '(', 'A'}),
			},
			want:    &sampleTrieComspecs[2],
			want1:   3,
			wantErr: false,
		},
		{
			name: "GS ( k",
			args: args{
				nodes: buildTrie(sampleTrieComspecs),
				r:     bytes.NewReader([]byte{byte(bGS), '(', 'k'}),
			},
			want:    &sampleTrieComspecs[3],
			want1:   3,
			wantErr: false,
		},
		{
			name: "ESC ( Y",
			args: args{
				nodes: buildTrie(sampleTrieComspecs),
				r:     bytes.NewReader([]byte{0x1B, '(', 'Y'}),
			},
			want:    &sampleTrieComspecs[7],
			want1:   3,
			wantErr: false,
		},
		{
			name: "not a control byte",
			args: args{
				nodes: buildTrie(sampleTrieComspecs),
				r:     bytes.NewReader([]byte{'A'}),
			},
			want:    nil,
			want1:   0,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := findComSpec(tt.args.nodes, tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("findComSpec() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("findComSpec() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("findComSpec() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
