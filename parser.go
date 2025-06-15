package senddat

import (
	"bufio"
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"text/scanner"
)

// ControlCode represents a control code used in the ESC/P, ESC/POS, and ESC/P2
// protocols.
type ControlCode byte

//go:generate stringer -type=ControlCode -trimprefix=b
const (
	bEOT ControlCode = 0x04 // End of Transmission character
	bBS  ControlCode = 0x08 // Backspace character
	bHT  ControlCode = 0x09 // Horizontal Tab character
	bLF  ControlCode = 0x0A // Line Feed character
	bFF  ControlCode = 0x0C // Form Feed character
	bCR  ControlCode = 0x0D // Carriage Return character
	bDLE ControlCode = 0x10 // Data Link Escape character
	bCAN ControlCode = 0x18 // Cancel character
	bESC ControlCode = 0x1B // Escape character
	bFS  ControlCode = 0x1C // File Separator character
	bGS  ControlCode = 0x1D // Group Separator character
	bSP  ControlCode = 0x20 // Space character
)

var tokenMap = map[string]ControlCode{
	bESC.String(): bESC,
	bGS.String():  bGS,
	bFS.String():  bFS,
	bCR.String():  bCR,
	bLF.String():  bLF,
	bFF.String():  bFF,
	bHT.String():  bHT,
	bDLE.String(): bDLE,
	bCAN.String(): bCAN,
	bBS.String():  bBS,
	bEOT.String(): bEOT,
	bSP.String():  bSP,
}

type errWriter struct {
	io.Writer
	Err error
	N   int
}

// Write implements io.Writer interface.
func (ew *errWriter) Write(p []byte) {
	if ew.Err != nil {
		return
	}
	n, err := ew.Writer.Write(p)
	if err != nil {
		ew.Err = err
		return
	}
	ew.N += n
}

func (ew *errWriter) Fprintf(format string, args ...any) {
	if ew.Err != nil {
		return
	}
	n, err := fmt.Fprintf(ew.Writer, format, args...)
	if err != nil {
		ew.Err = err
		return
	}
	ew.N += n
}

func Parse(w io.Writer, r io.Reader) error {

	var bw = bufio.NewWriter(w)
	defer bw.Flush()

	var ew = errWriter{Writer: bw}

	var s scanner.Scanner
	s.Init(r)
	s.Mode = scanner.ScanIdents | scanner.ScanStrings | scanner.ScanInts | scanner.ScanComments | scanner.ScanRawStrings
	s.Error = func(s *scanner.Scanner, msg string) {
		slog.Error("scanner", "error", msg, "line", s.Line, "pos", s.Pos())
	}
	for tok := s.Scan(); tok != scanner.EOF; tok = s.Scan() {
		t := s.TokenText()
		lg := slog.With("line", s.Line, "pos", s.Pos(), "value", t)
		switch tok {
		case scanner.Ident:
			lg.Debug("identifier")
			t := s.TokenText()
			if code, ok := tokenMap[t]; ok {
				ew.Write([]byte{byte(code)})
			} else {
				return fmt.Errorf("unknown identifier: %s at line %d, pos %v", t, s.Line, s.Pos())
			}
		case scanner.String:
			lg.Debug("string")
			text := strings.Trim(t, `"`)
			ew.Write([]byte(text))
		case scanner.RawString:
			lg.Debug("raw string")
			text := strings.Trim(t, "`")
			ew.Write([]byte(text))
		case scanner.Int:
			lg.Debug("integer")
			b, err := atob(t)
			if err != nil {
				return fmt.Errorf("invalid integer: %s at line %d, pos %v", t, s.Line, s.Pos())
			}
			ew.Write([]byte{b})
		case scanner.Char:
			lg.Info("char", "value", t, "line", s.Line, "pos", s.Pos())
		case scanner.Comment:
			lg.Debug("comment", "value", t, "line", s.Line, "pos", s.Pos())
		case sdDelayMs, sdKeyInput, sdPrint, sdComment, sdxInclude, sdxImage: // senddat command
			if err := senddatCommand(w, &s, tok); err != nil {
				return err
			}
		default:
			lg.Warn("unknown token", "code", tok, "token", scanner.TokenString(tok))
		}
	}
	if s.ErrorCount > 0 {
		return fmt.Errorf("parsing errors: %d", s.ErrorCount)
	}
	if ew.Err != nil {
		return fmt.Errorf("write error: %w", ew.Err)
	}
	return nil
}

// atob is similar to atoi but returns an 8-bit unsigned integer.
func atob(t string) (byte, error) {
	var scanfmt = "%d"
	if len(t) > 0 && t[0] == '0' && len(t) > 1 {
		t = strings.ToLower(t)
		switch t[1] {
		case 'x':
			// Hexadecimal integer
			scanfmt = "0x%x"
		case 'b':
			scanfmt = "0b%b"
		case 'o':
			scanfmt = "0o%o"
		}
	}
	var b byte
	_, err := fmt.Sscanf(t, scanfmt, &b)
	if err != nil {
		return 0, fmt.Errorf("scan error: %w", err)
	}
	return b, nil
}

// ParseString parses the string, like 'ESC "@"' and returns bytes.
func ParseString(s string) ([]byte, error) {
	var buf bytes.Buffer
	if err := Parse(&buf, strings.NewReader(s)); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
