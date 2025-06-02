package senddat

import (
	"bytes"
	_ "embed"
	"io"
	"testing"
)


func TestDecode(t *testing.T) {
	type args struct {
		r io.Reader
	}
	tests := []struct {
		name    string
		args    args
		wantW   string
		wantErr bool
	}{
		{
			name: "testESC",
			args: args{
				r: bytes.NewReader(loadTestFile(t, "testdata/POS/80x72.prn")),
			},
			wantW:   "",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &bytes.Buffer{}
			if err := Decode(w, tt.args.r); (err != nil) != tt.wantErr {
				t.Errorf("Decode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotW := w.String(); gotW != tt.wantW {
				t.Errorf("Decode() = %v, want %v", gotW, tt.wantW)
			}
		})
	}
}
