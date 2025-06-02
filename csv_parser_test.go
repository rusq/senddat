package senddat

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const sampleCSV = ` prefix,name,arg_names,payload_formula
"1B 40","Initialize Printer (ESC @)",,
"1B 2A","Bit Image Mode (ESC *)",m nL nH,"nL + 256 * nH"
"1B 4A","Print and feed paper (ESC J)",n,
"1B 61","Set Justification (ESC a)",n,
"1D 56","Cut Paper (GS V)",n,
`

func Test_readCommandSpecs(t *testing.T) {
	type args struct {
		r io.Reader
	}
	tests := []struct {
		name    string
		args    args
		want    []CommandSpec
		wantErr bool
	}{
		{
			name: "simple test",
			args: args{
				r: strings.NewReader(sampleCSV),
			},
			want: []CommandSpec{
				{
					Prefix:   []byte{byte(bESC), 0x40},
					Name:     "Initialize Printer (ESC @)",
					ArgCount: 0,
					ArgNames: []string{},
					payloadFn: func(args []byte) (int, error) {
						panic("should not be called")
					},
				},
				{
					Prefix:   []byte{byte(bESC), 0x2a},
					Name:     "Bit Image Mode (ESC *)",
					ArgCount: 3,
					ArgNames: []string{"m", "nL", "nH"},
					payloadFn: func(args []byte) (int, error) {
						panic("TODO")
					},
				},
				{
					Prefix:   []byte{0x1b, 0x4a},
					Name:     "Print and feed paper (ESC J)",
					ArgCount: 1,
					ArgNames: []string{"n"},
					payloadFn: func(args []byte) (int, error) {
						panic("TODO")
					},
				},
				{
					Prefix:   []byte{0x1b, 0x61},
					Name:     "Set Justification (ESC a)",
					ArgCount: 1,
					ArgNames: []string{"n"},
					payloadFn: func(args []byte) (int, error) {
						panic("TODO")
					},
				},
				{
					Prefix:   []byte{0x1d, 0x56},
					Name:     "Cut Paper (GS V)",
					ArgCount: 1,
					ArgNames: []string{"n"},
					payloadFn: func(args []byte) (int, error) {
						panic("TODO")
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := readCommandSpecs(tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("readCommandSpecs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.EqualExportedValues(t, got, tt.want)
		})
	}
}

func Test_makePayloadFn(t *testing.T) {
	type args struct {
		exprStr  string
		argNames []string
	}
	tests := []struct {
		name       string
		args       args
		payload    []byte
		wantRes    int
		wantResErr bool
		wantErr    bool
	}{
		{
			name:       "expression",
			args:       args{"mL+mH*256", []string{"n", "mL", "mH"}},
			payload:    []byte{128, 1, 1},
			wantRes:    257,
			wantResErr: false,
			wantErr:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := makePayloadFn(tt.args.exprStr, tt.args.argNames)
			if (err != nil) != tt.wantErr {
				t.Errorf("makePayloadFn() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != nil {
				if res, err := got(tt.payload); (err != nil) != tt.wantResErr {
					t.Errorf("result function error = %v, wantResErr %v", err, tt.wantResErr)
				} else {
					assert.Equal(t, tt.wantRes, res)
				}
			}
		})
	}
}

func TestCommandSpec_String(t *testing.T) {
	tests := []struct {
		name string
		cs   CommandSpec
		want string
	}{
		{
			name: "ESC @",
			cs: CommandSpec{
				Prefix: []byte{0x1b, '@'},
			},
			want: "ESC @ ",
		},
		{
			name: "GS (L",
			cs: CommandSpec{
				Prefix: []byte{0x1d, 0x28, 0x4c},
			},
			want: "GS ( L ",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cs.String(); got != tt.want {
				t.Errorf("CommandSpec.String() = %v, want %v", got, tt.want)
			}
		})
	}
}
