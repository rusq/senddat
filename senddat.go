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
	// standard senddat commands
	sdDelayMs  = '*'
	sdComment  = '\''
	sdKeyInput = '.'
	sdPrint    = '!'

	// extended senddat commands
	sdxInclude = '@' // include file
	sdxImage   = '#' // include image file
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
func senddatCommand(w io.Writer, s *scanner.Scanner, command rune) error {
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
	case sdxInclude:
		filename, err := readln(s, maxStrLen)
		if err != nil {
			return fmt.Errorf("error reading include filename at position %v: %w", s.Pos(), err)
		}
		slog.Debug("include file", "filename", filename, "line", s.Line, "pos", s.Pos())
		if n, err := copyfile(w, filename); err != nil {
			return fmt.Errorf("error including file '%s' at line %d, pos %v: %w", filename, s.Line, s.Pos(), err)
		} else {
			slog.Info("included file", "filename", filename, "bytes", n, "line", s.Line, "pos", s.Pos())
		}
	case sdxImage:
		slog.Warn("image inclusion is not supported yet", "line", s.Line, "pos", s.Pos())
		// TODO establish a protocol for image inclusion, i.e. how to handle the image data.
		_, err := readln(s, maxStrLen)
		if err != nil {
			return fmt.Errorf("error reading image filename at position %v: %w", s.Pos(), err)
		}
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

// copyfile copies the contents of the file with the given filename to the writer w.
// It returns the number of bytes copied and an error if any.
func copyfile(w io.Writer, filename string) (int64, error) {
	f, err := os.Open(filename)
	if err != nil {
		return 0, fmt.Errorf("error opening file '%s': %w", filename, err)
	}
	defer f.Close()
	n, err := io.Copy(w, f)
	if err != nil {
		return n, fmt.Errorf("error copying file '%s': %w", filename, err)
	}
	return n, nil
}
