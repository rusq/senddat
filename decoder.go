package senddat

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
)

type DecodeError struct {
	Message string
	Offset  int
	Err     error
}

func (e *DecodeError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s at offset %d: %s", e.Message, e.Offset, e.Err)
	}
	return fmt.Sprintf("%s at offset %d", e.Message, e.Offset)
}

func (e *DecodeError) Unwrap() error {
	return e.Err
}

func Decode(w io.Writer, r io.Reader) error {
	// Create a new parser instance
	var decerr = func(offset int, msg string, err ...error) error {
		if len(err) > 0 {
			return &DecodeError{Message: msg, Offset: offset, Err: err[0]}
		}
		return &DecodeError{Message: msg, Offset: offset}
	}
	var ew = errWriter{Writer: w}

	p := NewParser(r)

LOOP:
	for {
		if ew.Err != nil {
			return ew.Err
		}

		cmd, err := p.NextCommand()
		if err != nil {
			if err == io.EOF {
				break LOOP // End of stream
			}
			return decerr(p.pos, "failed to read command", err)
		}
		if cmd == nil {
			continue // Skip nil commands
		}
		slog.Debug("command", "offset", cmd.Offset, "name", cmd.Name, "args", cmd.Args)
	}

	return nil
}

type Command struct {
	Offset int    // Position in the input stream
	Bytes  []byte // Raw bytes of the command
	Name   string // Decoded command name
	Args   []byte // Optional arguments
}

type Parser struct {
	r     *bufio.Reader
	pos   int // Running byte offset in the stream
	limit int // Optional read limit (0 = no limit)
}

func NewParser(r io.Reader) *Parser {
	return &Parser{
		r: bufio.NewReader(r),
	}
}

// Helper: read exactly one byte
func (p *Parser) readByte() (byte, error) {
	b, err := p.r.ReadByte()
	if err == nil {
		p.pos++
	}
	return b, err
}

func (p *Parser) NextCommand() (*Command, error) {
	startPos := p.pos
	b, err := p.readByte()
	if err != nil {
		return nil, err
	}

	switch b {
	case 0x1B: // ESC
		second, err := p.readByte()
		if err != nil {
			return nil, io.ErrUnexpectedEOF
		}

		switch second {
		case '@':
			return &Command{
				Offset: startPos,
				Bytes:  []byte{0x1B, second},
				Name:   "Initialize Printer (ESC @)",
			}, nil

		case 'J': // Print and feed paper
			n, err := p.readByte()
			if err != nil {
				return nil, io.ErrUnexpectedEOF
			}
			return &Command{
				Offset: startPos,
				Bytes:  []byte{0x1B, second, n},
				Name:   fmt.Sprintf("Print and Feed Paper (ESC J = %d)", n),
				Args:   []byte{n},
			}, nil

		case 'U': // Enable unidirectional printing
			n, err := p.readByte()
			if err != nil {
				return nil, io.ErrUnexpectedEOF
			}
			return &Command{
				Offset: startPos,
				Bytes:  []byte{0x1B, second, n},
				Name:   fmt.Sprintf("Set Unidirectional Printing (ESC U = %d)", n),
				Args:   []byte{n},
			}, nil

		case 'a':
			align, err := p.readByte()
			if err != nil {
				return nil, io.ErrUnexpectedEOF
			}
			return &Command{
				Offset: startPos,
				Bytes:  []byte{0x1B, second, align},
				Name:   fmt.Sprintf("Set Justification (ESC a = %d)", align),
				Args:   []byte{align},
			}, nil

		case 'd': // print and feed n lines
			n, err := p.readByte()
			if err != nil {
				return nil, io.ErrUnexpectedEOF
			}
			return &Command{
				Offset: startPos,
				Bytes:  []byte{0x1B, second, n},
				Name:   fmt.Sprintf("Print and Feed %d Lines (ESC d = %d)", n, n),
				Args:   []byte{n},
			}, nil

		case '*': // Bit image mode
			mode, err := p.readByte()
			if err != nil {
				return nil, io.ErrUnexpectedEOF
			}

			nL, err := p.readByte()
			if err != nil {
				return nil, io.ErrUnexpectedEOF
			}

			nH, err := p.readByte()
			if err != nil {
				return nil, io.ErrUnexpectedEOF
			}

			dataLen := int(nL) + int(nH)*256
			data := make([]byte, dataLen)
			if _, err := io.ReadFull(p.r, data); err != nil {
				return nil, fmt.Errorf("incomplete ESC * image payload: %w", err)
			}
			p.pos += dataLen

			full := append([]byte{0x1B, second, mode, nL, nH}, data...)
			return &Command{
				Offset: startPos,
				Bytes:  full,
				Name:   fmt.Sprintf("Bit Image Mode (ESC * m=%d, n=%d)", mode, dataLen),
				Args:   append([]byte{mode, nL, nH}, data...),
			}, nil

		default:
			return &Command{
				Offset: startPos,
				Bytes:  []byte{0x1B, second},
				Name:   fmt.Sprintf("Unknown ESC command: %[1]c 0x%[1]X", second),
			}, nil
		}

	case 0x1D: // GS
		second, err := p.readByte()
		if err != nil {
			return nil, io.ErrUnexpectedEOF
		}

		switch second {
		case 'V':
			mode, err := p.readByte()
			if err != nil {
				return nil, io.ErrUnexpectedEOF
			}
			return &Command{
				Offset: startPos,
				Bytes:  []byte{0x1D, second, mode},
				Name:   fmt.Sprintf("Cut Paper (GS V = %d)", mode),
				Args:   []byte{mode},
			}, nil

		default:
			return &Command{
				Offset: startPos,
				Bytes:  []byte{0x1D, second},
				Name:   fmt.Sprintf("Unknown GS command: 0x%X", second),
			}, nil
		}

	case '\n':
		return &Command{
			Offset: startPos,
			Bytes:  []byte{'\n'},
			Name:   "Line Feed",
		}, nil

	default:
		if b >= 0x20 && b <= 0x7E {
			return &Command{
				Offset: startPos,
				Bytes:  []byte{b},
				Name:   "Printable ASCII",
			}, nil
		}
		return &Command{
			Offset: startPos,
			Bytes:  []byte{b},
			Name:   fmt.Sprintf("Raw Byte 0x%02X", b),
		}, nil
	}
}


