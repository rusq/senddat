package senddat

import (
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
	bESC ControlCode = 0x1B // Escape character
	bGS  ControlCode = 0x1D // Group Separator character
	bFS  ControlCode = 0x1C // File Separator character
	bCR  ControlCode = 0x0D // Carriage Return character
	bLF  ControlCode = 0x0A // Line Feed character
	bFF  ControlCode = 0x0C // Form Feed character
	bBS  ControlCode = 0x08 // Backspace character
	bHT  ControlCode = 0x09 // Horizontal Tab character
	bDLE ControlCode = 0x10 // Data Link Escape character
	bCAN ControlCode = 0x18 // Cancel character
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
	var s scanner.Scanner
	var ew = errWriter{Writer: w}
	s.Init(r)
	s.Mode = scanner.ScanIdents | scanner.ScanStrings | scanner.ScanInts | scanner.ScanComments
	s.Error = func(s *scanner.Scanner, msg string) {
		slog.Error("scanner", "error", msg, "line", s.Line, "pos", s.Pos())
	}
	for tok := s.Scan(); tok != scanner.EOF; tok = s.Scan() {
		switch tok {
		case scanner.Ident:
			slog.Debug("identifier", "text", s.TokenText(), "line", s.Line, "pos", s.Pos())
			t := s.TokenText()
			if code, ok := tokenMap[t]; ok {
				ew.Write([]byte{byte(code)})
			} else {
				return fmt.Errorf("unknown identifier: %s at line %d, pos %v", s.TokenText(), s.Line, s.Pos())
			}
		case scanner.String:
			slog.Debug("string", "value", s.TokenText(), "line", s.Line, "pos", s.Pos())
			text := strings.Trim(s.TokenText(), `"`)
			ew.Write([]byte(text))
		case scanner.Int:
			t := s.TokenText()
			slog.Debug("integer", "value", t, "line", s.Line, "pos", s.Pos())
			// Convert integer to byte and write it
			b, err := atob(t)
			if err != nil {
				return fmt.Errorf("invalid integer: %s at line %d, pos %v", t, s.Line, s.Pos())
			}
			ew.Write([]byte{b})
		case scanner.Char:
			slog.Info("char", "value", s.TokenText(), "line", s.Line, "pos", s.Pos())
		case scanner.Comment:
			slog.Debug("comment", "value", s.TokenText(), "line", s.Line, "pos", s.Pos())
		case sdDelayMs, sdKeyInput, sdPrint, sdComment: // senddat delay
			if err := senddatCommand(&s, tok); err != nil {
				return err
			}
		default:
			slog.Warn("unknown token", "code", tok, "token", scanner.TokenString(tok), "text", s.TokenText(), "line", s.Line, "pos", s.Pos())
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
