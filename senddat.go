package senddat

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"text/scanner"
	"time"
)

// senddat commands
const (
	sdDelayMs  = '*'
	sdComment  = '\''
	sdKeyInput = '.'
	sdPrint    = '!'
)

// maxStrLen is the maximum string length for the readln
const maxStrLen = 250

var (
	// SenddatOutput is the default senddat output stream
	SenddatOutput = os.Stderr

	gWaitMultiplier time.Duration = 1 // senddat wait multiplier, to turn it off in tests.
)

var errTooLong = errors.New("string length exceeded")

// senddatCommand is a senddat command executor.
func senddatCommand(s *scanner.Scanner, command rune) error {
	switch command {
	case sdComment:
		slog.Debug("start of the comment line", "text", s.TokenText(), "line", s.Line, "pos", s.Pos())
	case sdDelayMs:
		val := s.Scan()
		if val != scanner.Int {
			return fmt.Errorf("expected integer after '*', got '%s' at line %d, pos %v", s.TokenText(), s.Line, s.Pos())
		}
		t := s.TokenText()
		slog.Debug("delay", "text", t, "line", s.Line, "pos", s.Pos())
		ms, err := strconv.Atoi(t)
		if err != nil {
			return fmt.Errorf("invalid delay value: %s at line %d, pos %v", t, s.Line, s.Pos())
		}
		slog.Debug("delay value", "ms", ms, "line", s.Line, "pos", s.Pos())
		time.Sleep(time.Duration(ms) * time.Millisecond * gWaitMultiplier)
	case sdKeyInput:
		msg, err := readln(s, maxStrLen)
		if err != nil {
			return fmt.Errorf("error at position %v: %w", s.Pos(), err)
		}
		fmt.Fprintln(SenddatOutput, msg)
		// fmt.Scanln()
	case sdPrint:
		msg, err := readln(s, maxStrLen)
		if err != nil {
			return fmt.Errorf("error at position %v: %w", s.Pos(), err)
		}
		fmt.Fprintln(SenddatOutput, msg)
	default:
		return fmt.Errorf("unhandled senddat command: '%c'", command)
	}
	return nil
}

// readln consumes the line of text from the current postition until CR
// character reading at maximum n runes. If n is reached, it returns the data
// read so far and errTooLong error.
func readln(s *scanner.Scanner, n int) (string, error) {
	var buf strings.Builder
	for range n {
		ch := s.Next()
		if ch == scanner.EOF {
			return "", fmt.Errorf("scanln: %w at %v", io.ErrUnexpectedEOF, s.Pos())
		}
		if ch == '\n' {
			return buf.String(), nil
		}
		if _, err := buf.WriteRune(ch); err != nil {
			return buf.String(), err
		}
	}
	return buf.String(), errTooLong
}
