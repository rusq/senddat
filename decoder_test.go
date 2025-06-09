package senddat

import (
	"bytes"
	_ "embed"
	"io"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	genericComspecs, loadErr = loadCommandSpecs("drivers/escpos-3.40.csv")
)

func init() {
	if loadErr != nil {
		panic(loadErr)
	}
}

var (
	csLF        = genericComspecs[13]
	csEmphasis  = genericComspecs[4]
	csUnderline = genericComspecs[1]
	csCharfont  = genericComspecs[10]
)

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
				r:        bytes.NewReader(toPRN(t, exampleFS, "examples/POS/weight.dat")),
				comspecs: genericComspecs,
			},
			want: []Entry{
				{Offset: 0, Data: []byte("*** font weight test ****")},
				{Offset: 25, Spec: &csLF},
				{Offset: 26, Spec: &csEmphasis, Args: []byte{1}},
				{Offset: 29, Data: []byte("Emphasized mode")},
				{Offset: 44, Spec: &csEmphasis, Args: []byte{0}},
				{Offset: 47, Spec: &csLF},
				{Offset: 48, Spec: &csUnderline, Args: []byte{1}},
				{Offset: 51, Data: []byte("Underline")},
				{Offset: 60, Spec: &csUnderline, Args: []byte{0}},
				{Offset: 63, Data: []byte(" off")},
				{Offset: 67, Spec: &csLF},
				{Offset: 68, Data: []byte("*** font height and width ***")},
				{Offset: 97, Spec: &csLF},
				{Offset: 98, Spec: &csCharfont, Args: []byte{17}},
				{Offset: 101, Data: []byte("Double H + W")},
				{Offset: 113, Spec: &csLF},
				{Offset: 114, Spec: &csCharfont, Args: []byte{16}},
				{Offset: 117, Data: []byte("Double Width")},
				{Offset: 129, Spec: &csLF},
				{Offset: 130, Spec: &csCharfont, Args: []byte{1}},
				{Offset: 133, Data: []byte("Double Height")},
				{Offset: 146, Spec: &csLF},
				{Offset: 147, Spec: &csCharfont, Args: []byte{0}},
				{Offset: 150, Data: []byte("back to normal")},
				{Offset: 164, Spec: &csLF},
			},
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
