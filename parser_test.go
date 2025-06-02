package senddat

import (
	"bytes"
	"embed"
	_ "embed"
	"io"
	"strings"
	"testing"
)

const (
	testESC    = `ESC "p" 0 2 20`
	testGS     = `GS "(L" 139   7  48  67  48   "G1"   1 128   0 120   0  49`
	testCRLF   = `"Hello, World!" CR LF`
	testBinary = `0xFF 0xFF 0xFF 0xFF 0xFF 0xFF 0xFF 0xFF 0xFF 0xFF 0xFF 0xFF 0xFF 0xFF 0xFF 0xFF`
)

//go:embed testdata/POS/*.dat testdata/POS/*.prn
var testFS embed.FS

func loadTestFile(t *testing.T, name string) []byte {
	t.Helper()
	data, err := testFS.ReadFile(name)
	if err != nil {
		t.Fatalf("Failed to read test file %s: %v", name, err)
	}
	return data
}

func TestParse(t *testing.T) {
	type args struct {
		r io.Reader
	}
	tests := []struct {
		name    string
		args    args
		wantW   []byte
		wantErr bool
	}{
		{
			name: "testESC",
			args: args{
				r: strings.NewReader(testESC),
			},
			wantW:   []byte{0x1B, 'p', 0, 2, 20},
			wantErr: false,
		},
		{
			name: "testGS",
			args: args{
				r: strings.NewReader(testGS),
			},
			wantW:   []byte{0x1D, '(', 'L', 139, 7, 48, 67, 48, 'G', '1', 1, 128, 0, 120, 0, 49},
			wantErr: false,
		},
		{
			name: "testCRLF",
			args: args{
				r: strings.NewReader(testCRLF),
			},
			wantW:   []byte("Hello, World!\r\n"),
			wantErr: false,
		},
		{
			name: "testBinary",
			args: args{
				r: strings.NewReader(testBinary),
			},
			wantW:   []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
			wantErr: false,
		},
		{
			name: "test graphics.dat",
			args: args{
				r: bytes.NewReader(loadTestFile(t, "testdata/POS/graphics.dat")),
			},
			wantW:   loadTestFile(t, "testdata/POS/graphics.prn"),
			wantErr: false,
		},
		{
			name: "test label.dat",
			args: args{
				r: bytes.NewReader(loadTestFile(t, "testdata/POS/label.dat")),
			},
			wantW:   loadTestFile(t, "testdata/POS/label.prn"),
			wantErr: false,
		},
		{
			name: "senddat delay",
			args: args{
				r: strings.NewReader("*256\n\"abc\""),
			},
			wantW:   []byte("abc"),
			wantErr: false,
		},
		{
			name: "print command",
			args: args{
				r: strings.NewReader("!hello\n0 1 2"),
			},
			wantW:   []byte{0, 1, 2},
			wantErr: false,
		},
		{
			name: "any key command",
			args: args{
				r: strings.NewReader(".Press any key to continue\n4 5 6"),
			},
			wantW:   []byte{4, 5, 6},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &bytes.Buffer{}
			if err := Parse(w, tt.args.r); (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotW := w.Bytes(); !bytes.Equal(gotW, tt.wantW) {
				t.Errorf("Parse() = % x, want % x", gotW, tt.wantW)
			}
		})
	}
}

func Test_atob(t *testing.T) {
	type args struct {
		t string
	}
	tests := []struct {
		name    string
		args    args
		want    byte
		wantErr bool
	}{
		{
			name: "valid decimal",
			args: args{
				t: "255",
			},
			want:    255,
			wantErr: false,
		},
		{
			name: "valid hex",
			args: args{
				t: "0xFF",
			},
			want:    255,
			wantErr: false,
		},
		{
			name: "valid octal",
			args: args{
				t: "0o377",
			},
			want:    255,
			wantErr: false,
		},
		{
			name: "valid binary",
			args: args{
				t: "0b11111111",
			},
			want:    255,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := atob(tt.args.t)
			if (err != nil) != tt.wantErr {
				t.Errorf("atob() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("atob() = %v, want %v", got, tt.want)
			}
		})
	}
}
