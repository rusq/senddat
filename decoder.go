package senddat

import (
	"bufio"
	"errors"
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

// Decode decodes a stream of PRN commands from the provided reader using the
// provided command specifications. It returns a slice of Command structs or an
// error if decoding fails.
func Decode(r io.Reader, spec []CommandSpec) ([]Command, error) {
	// Create a new parser instance
	p, err := NewParser(r, spec)
	if err != nil {
		return nil, fmt.Errorf("failed to create parser: %w", err)
	}

	var decerr = func(offset int, msg string, err ...error) error {
		if len(err) > 0 {
			return &DecodeError{Message: msg, Offset: offset, Err: err[0]}
		}
		return &DecodeError{Message: msg, Offset: offset}
	}

	var commands []Command
LOOP:
	for {
		cmd, err := p.NextCommand2()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break LOOP // End of stream
			}
			return nil, decerr(p.pos, "failed to read command", err)
		}
		if cmd == nil {
			continue // Skip nil commands
		}
		commands = append(commands, *cmd)
		slog.Debug("command", "offset", cmd.Offset, "name", cmd.Name(), "args", cmd.Args)
	}

	return commands, nil
}

type Command struct {
	// Position in the input stream
	Offset int
	// Spec is the command specification.
	Spec  *CommandSpec
	Bytes []byte // Raw bytes, if it's not a known command
	// Args is the optional arguments for commands that have them. i.e. for ESC
	// J n, will contain the value of n
	Args []byte
	// Payload is the optional payload data for commands that have it, like:
	// ESC * m nL nH data
	Payload []byte
}

func (c *Command) Name() string {
	if c.Spec != nil {
		return c.Spec.Name
	}
	if len(c.Bytes) == 0 {
		return fmt.Sprintf("Raw Bytes 0x%02X", c.Bytes[0])
	}
	return "INVALID COMMAND"
}

type Parser struct {
	r     *bufio.Reader
	cst   *trieNode // Trie for command specs
	pos   int       // Running byte offset in the stream
	limit int       // Optional read limit (0 = no limit)
}

func NewParser(r io.Reader, spec []CommandSpec) (*Parser, error) {
	return &Parser{
		r:   bufio.NewReader(r),
		cst: buildTrie(spec),
	}, nil
}

// Helper: read exactly one byte
func (p *Parser) readByte() (byte, error) {
	b, err := p.r.ReadByte()
	if err == nil {
		p.pos++
	}
	return b, err
}

func (p *Parser) readBytes(n int) ([]byte, error) {
	if n < 0 {
		return nil, fmt.Errorf("invalid read length: %d", n)
	} else if n == 0 {
		return nil, nil // No bytes to read
	}

	data := make([]byte, n)
	if _, err := io.ReadFull(p.r, data); err != nil {
		return nil, fmt.Errorf("failed to read %d bytes at position %d: %w", n, p.pos, err)
	}
	p.pos += n
	return data, nil
}

func (p *Parser) UnreadByte() error {
	if err := p.r.UnreadByte(); err != nil {
		return fmt.Errorf("failed to unread byte at position %d: %w", p.pos, err)
	}
	p.pos--
	return nil
}

func (p *Parser) ReadByte() (byte, error) {
	return p.readByte()
}

func (p *Parser) NextCommand2() (*Command, error) {
	startPos := p.pos
	cs, found, err := findComSpec(p.cst, p)
	if err != nil {
		return nil, err
	}
	if !found {
		b, err := p.readByte()
		if err != nil {
			return nil, err
		}
		// If we can't find a command spec, treat it as a raw byte command
		return &Command{
			Offset: startPos,
			Bytes:  []byte{b},
		}, nil
	}

	args, err := p.readBytes(cs.ArgCount)
	if err != nil {
		return nil, fmt.Errorf("failed to read command args at position %d: %w", startPos, err)
	}
	if len(args) < cs.ArgCount {
		return nil, fmt.Errorf("expected %d args for command %s, got %d at position %d", cs.ArgCount, cs.Name, len(args), startPos)
	}
	var payload []byte
	if cs.payloadFn != nil {
		payloadLen, err := cs.payloadFn(args)
		if err != nil {
			return nil, fmt.Errorf("failed to execute payload function for command %s at position %d: %w", cs.Name, startPos, err)
		}
		if payloadLen > 0 {
			payload, err = p.readBytes(payloadLen)
			if err != nil {
				return nil, fmt.Errorf("failed to read payload for command %s at position %d: %w", cs.Name, startPos, err)
			}
			if len(payload) < payloadLen {
				return nil, fmt.Errorf("expected %d bytes of payload for command %s, got %d at position %d", payloadLen, cs.Name, len(payload), startPos)
			}
		}
	}

	var c = &Command{
		Offset:  startPos,
		Spec:    cs,
		Args:    args,
		Payload: payload,
	}
	return c, nil
}

var errUnhandled = errors.New("unhandled command prefix")

func findComSpec(n *trieNode, r io.ByteScanner) (*CommandSpec, bool, error) {
	current := n
	for depth := 0; ; depth++ {
		b, err := r.ReadByte()
		if err != nil {
			return nil, false, err // Error reading byte
		}

		if nextNode, exists := current.children[b]; exists {
			current = nextNode
			if current.spec != nil {
				return current.spec, true, nil // Found a command spec
			}
		} else {
			if depth > 0 {
				return nil, false, errUnhandled // Unhandled command prefix
			}
			if err := r.UnreadByte(); err != nil {
				return nil, false, err // Error unread byte
			}
			return nil, false, nil // No matching command found
		}
	}
}
