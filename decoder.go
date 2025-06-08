package senddat

import (
	"bufio"
	"bytes"
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
func Decode(r io.Reader, spec []CommandSpec) ([]Entry, error) {
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

	var commands []Entry
	var ignore bool
LOOP:
	for {
		cmd, err := p.Next(ignore)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break LOOP // End of stream
			}
			return nil, decerr(p.pos, "failed to read command", err)
		}
		if cmd == nil {
			continue // Skip nil commands
		}
		ignore = false // Reset ignore flag for the next command
		if cmd.IsCommand() && cmd.Spec.Ignore {
			// to ignore false positives in the payload following the ignored command,
			// we set the ignore flag for unknown commands.
			// TODO: test this.
			ignore = true
			slog.Debug("ignoring command", "offset", cmd.Offset, "name", cmd.Name())
			continue // Skip this command
		}
		commands = append(commands, *cmd)
		slog.Debug("command", "offset", cmd.Offset, "name", cmd.Name(), "args", cmd.Args)
	}

	return commands, nil
}

// Entry represents a single command entry in the PRN stream.
type Entry struct {
	// Position in the input stream where the command or data starts.
	Offset int
	// Spec is the command specification.
	Spec *CommandSpec
	// Data is the raw bytes read from the stream. If the command is known,
	// it will be empty.
	Data []byte
	// Args is the optional arguments for commands that have them. i.e. for ESC
	// J n, will contain the value of n
	Args []byte
	// Payload is the optional payload data for commands that have it, like:
	// ESC * m nL nH data
	Payload []byte
}

func (c Entry) IsCommand() bool {
	return c.Spec != nil && len(c.Data) == 0
}

func (c Entry) IsRaw() bool {
	return c.Spec == nil && len(c.Data) > 0
}

func (c Entry) IsEmpty() bool {
	return c.Spec == nil && len(c.Data) == 0
}

func (c Entry) Name() string {
	if c.Spec != nil {
		return c.Spec.Name
	}
	if len(c.Data) == 0 {
		return fmt.Sprintf("Raw Bytes 0x%02X", c.Data[0])
	}
	return "INVALID COMMAND"
}

func (c Entry) String() string {
	if c.IsEmpty() {
		return fmt.Sprintf("[@%6d: EMPTY]", c.Offset)
	}
	var buf bytes.Buffer
	if c.IsRaw() {
		return fmt.Sprintf("[@%6d: RAW,len=%d %q]", c.Offset, len(c.Data), c.Data)
	}
	fmt.Fprintf(&buf, "[@%6d: %s", c.Offset, c.Name())
	if len(c.Args) == 0 {
		return buf.String() + "]"
	}
	args, err := c.Spec.ArgValues(c.Args)
	if err != nil {
		return fmt.Sprintf("[@%6d:ERROR %s]", c.Offset, err)
	}
	var argv []string
	for name, value := range args {
		argv = append(argv, fmt.Sprintf("%s=%d", name, value))
	}
	buf.WriteString(", args=")
	buf.WriteString(fmt.Sprintf("%v", argv))
	if len(c.Payload) > 0 {
		buf.WriteString(fmt.Sprintf(", payload=%d bytes", len(c.Payload)))
	}
	buf.WriteString("]")
	return buf.String()

}

type Interpreter struct {
	r     *bufio.Reader
	cst   *trieNode // Trie for command specs
	pos   int       // Running byte offset in the stream
	limit int       // Optional read limit (0 = no limit)
}

func NewParser(r io.Reader, spec []CommandSpec) (*Interpreter, error) {
	return &Interpreter{
		r:   bufio.NewReader(r),
		cst: buildTrie(spec),
	}, nil
}

// Helper: read exactly one byte
func (p *Interpreter) readByte() (byte, error) {
	b, err := p.r.ReadByte()
	if err == nil {
		p.pos++
	}
	return b, err
}

func (p *Interpreter) readBytes(n int) ([]byte, error) {
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

func (p *Interpreter) UnreadByte() error {
	if err := p.r.UnreadByte(); err != nil {
		return fmt.Errorf("failed to unread byte at position %d: %w", p.pos, err)
	}
	p.pos--
	return nil
}

func (p *Interpreter) ReadByte() (byte, error) {
	return p.readByte()
}

// Next reads the next command or raw data entry from the input stream.
// It returns an Entry containing the command specification, arguments, and
// payload if applicable, or raw data if no command is recognized.
// If ignoreUnknown is true, it will skip unknown commands and continue reading.
func (p *Interpreter) Next(ignoreUnknown bool) (*Entry, error) {
	startPos := p.pos
	var accum bytes.Buffer
	for {
		// Peek the next byte to determine if it's a command or raw data
		peeked, err := p.r.Peek(1)
		if err != nil {
			if errors.Is(err, io.EOF) {
				if accum.Len() > 0 {
					// If we have accumulated bytes, return them as a raw entry
					return &Entry{
						Offset: startPos,
						Data:   accum.Bytes(),
					}, nil
				}
				return nil, io.EOF // End of stream, no more data to read
			}
			return nil, fmt.Errorf("failed to peek byte at position %d: %w", startPos, err)
		}
		// if it's a command
		if _, nextIsCommand := p.cst.findChild(peeked[0]); !nextIsCommand {
			// If the first byte is not a command prefix, accumulate it as raw data
			b, err := p.readByte()
			if err != nil {
				return nil, fmt.Errorf("failed to read byte at position %d: %w", startPos, err)
			}
			if err := accum.WriteByte(b); err != nil {
				return nil, fmt.Errorf("failed to write byte %02X at position %d: %w", peeked[0], startPos, err)
			}
			continue // Continue accumulating bytes
		}
		// next is a command.
		if accum.Len() > 0 {
			// If we have accumulated bytes, return them as a raw entry
			return &Entry{
				Offset: startPos,
				Data:   accum.Bytes(),
			}, nil // Return accumulated bytes as a raw command
		}
		cs, _, err := findComSpec(p.cst, p)
		if err != nil {
			if errors.Is(err, errUnhandled) && ignoreUnknown {
				// If we encounter an unhandled command and ignoring unknown commands,
				// we can skip it and continue reading.
				slog.Debug("unhandled command", "offset", startPos, "byte", peeked[0])
				// Unread the byte so we can read the next command
				if err := p.UnreadByte(); err != nil {
					return nil, fmt.Errorf("failed to unread byte at position %d: %w", startPos, err)
				}
				continue // Skip this command and continue reading
			}
			return nil, err
		}
		if cs == nil {
			// pretty much impossible at this point
			return nil, fmt.Errorf("no command spec found at position %d", startPos)
		}
		cmd, err := p.readCommand(cs)
		if err != nil {
			return nil, fmt.Errorf("failed to read entry %s at position %d: %w", cs.Name, startPos, err)
		}
		cmd.Offset = startPos
		return cmd, nil
	}
}

func (p *Interpreter) readCommand(cs *CommandSpec) (*Entry, error) {
	if cs.ArgCount == 0 {
		return &Entry{
			Spec: cs,
		}, nil
	}

	// Read the command arguments
	args, err := p.readBytes(cs.ArgCount)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to read command args: %w", err)
	}
	if len(args) < cs.ArgCount {
		return nil, fmt.Errorf("expected %d args for command %s, got %d", cs.ArgCount, cs.Name, len(args))
	}
	var payload []byte
	if cs.payloadFn == nil {
		return &Entry{
			Spec: cs,
			Args: args,
		}, nil // No payload function, just return args
	}

	// Execute the payload function to determine the length of the payload
	// and read the payload data if necessary.
	payloadLen, err := cs.payloadFn(args)
	if err != nil {
		return nil, fmt.Errorf("failed to execute payload function for command %s at: %w", cs.Name, err)
	}
	if payloadLen > 0 {
		payload, err = p.readBytes(payloadLen)
		if err != nil {
			return nil, fmt.Errorf("failed to read payload for command %s: %w", cs.Name, err)
		}
		if len(payload) < payloadLen {
			return nil, fmt.Errorf("expected %d bytes of payload for command %s, got %d", payloadLen, cs.Name, len(payload))
		}
	}

	var c = &Entry{
		Spec:    cs,
		Args:    args,
		Payload: payload,
	}
	return c, nil
}

var errUnhandled = errors.New("unhandled command")

// findComSpec traverses the trie to find a command specification based on the
// bytes read from the provided ByteScanner. It returns the CommandSpec if found,
// an integer indicating number of bytes read from reader , and an error if any
// occurred during reading.
func findComSpec(n *trieNode, r io.ByteScanner) (*CommandSpec, int, error) {
	current := n
	read := 0 // Number of bytes read from the reader
	for depth := 0; ; depth++ {
		b, err := r.ReadByte()
		if err != nil {
			return nil, read, err // Error reading byte
		}
		read++ // Increment the read count

		if nextNode, exists := current.children[b]; exists {
			current = nextNode
			// I checked the ESC/POS spec and it doesn't seem that commands
			// that have n bytes are ever used in n-1 byte form, so we can
			// safely return the spec if it exists, as it guarantees that
			// depth+1 does not exist.
			// TODO: validate once the full spec is compiled.
			if current.spec != nil {
				return current.spec, read, nil // Found a command spec
			}
		} else {
			if depth > 0 {
				return nil, read, fmt.Errorf("%w: %x (\"%c\")", errUnhandled, b, b) // Unhandled command prefix
			}
			if err := r.UnreadByte(); err != nil {
				return nil, read, err // Error unread byte
			}
			read--                // Decrement the read count since we unread the byte
			return nil, read, nil // No matching command found
		}
	}
}
