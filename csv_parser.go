package senddat

import (
	"bytes"
	_ "embed"
	"encoding/csv"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"strconv"
	"strings"
)

//go:embed drivers/generic.csv
var genericcsv []byte

var GenericCommandSpecs []CommandSpec

func init() {
	var err error
	GenericCommandSpecs, err = readCommandSpecs(bytes.NewReader(genericcsv), defParseFn)
	if err != nil {
		panic(fmt.Sprintf("failed to load generic command specs: %v", err))
	}
}

type CommandSpec struct {
	Prefix      []byte
	Name        string
	ArgCount    int
	ArgNames    []string
	payloadFn   func(args []byte) (int, error)
	subcommands map[string]string // key: hex string of subcommand bytes
}

func (cs CommandSpec) String() string {
	var buf strings.Builder
	fmt.Fprintf(&buf, "%s ", ControlCode(cs.Prefix[0]))
	for _, ch := range cs.Prefix[1:] {
		fmt.Fprintf(&buf, "%c ", ch)
	}
	return buf.String()
}

// ArgValues returns a map of argument names to their values based on the provided args.
func (cs CommandSpec) ArgValues(args []byte) (map[string]uint8, error) {
	if len(args) != cs.ArgCount {
		return nil, fmt.Errorf("expected %d args, got %d", cs.ArgCount, len(args))
	}

	argValues := make(map[string]uint8)
	for i, name := range cs.ArgNames {
		if i < len(args) {
			argValues[name] = args[i]
		} else {
			argValues[name] = 0 // Default to 0 if no value provided
		}
	}
	return argValues, nil
}

func LoadCommandSpecsWithSubcommands(cmdCSV, subCSV string) ([]CommandSpec, error) {
	cmds, err := loadCommandSpecs(cmdCSV)
	if err != nil {
		return nil, err
	}

	subMap, err := loadSubcommands(subCSV)
	if err != nil {
		return nil, err
	}

	for i := range cmds {
		key := hexKey(cmds[i].Prefix)
		if subs, ok := subMap[key]; ok {
			cmds[i].subcommands = subs
		}
	}

	return cmds, nil
}

type parseFunc func(s string) ([]byte, error)

// defParseFn is the default parsing function for commands.  There are two
// currently to choose from:
//  1. ParseString - parses expressions like `ESC "@"'
//  2. parseHexBytes - parses hex bytes, i.e. `1B 40'
var defParseFn parseFunc = ParseString

func loadCommandSpecs(csvPath string) ([]CommandSpec, error) {
	f, err := os.Open(csvPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return readCommandSpecs(f, ParseString)
}

func readCommandSpecs(r io.Reader, parseFn parseFunc) ([]CommandSpec, error) {
	cr := csv.NewReader(r)
	cr.TrimLeadingSpace = true
	header, err := cr.Read()
	if err != nil {
		return nil, err
	}

	var specs []CommandSpec
	for {
		row, err := cr.Read()
		if errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return nil, err
		}

		// header is expected to be: "prefix,name,arg_names,payload_formula"
		rowMap := map[string]string{}
		for i, key := range header {
			rowMap[key] = row[i]
		}

		prefix, err := parseFn(rowMap["prefix"])
		if err != nil {
			return nil, fmt.Errorf("prefix %q: %v", rowMap["prefix"], err)
		}

		argNames := strings.Split(rowMap["arg_names"], " ")
		if argNames[0] == "" {
			argNames = []string{}
		}

		payloadFn, err := makePayloadFn(rowMap["payload_formula"], argNames)
		if err != nil {
			return nil, fmt.Errorf("payload fn for %x: %v", prefix, err)
		}

		specs = append(specs, CommandSpec{
			Prefix:    prefix,
			Name:      rowMap["name"],
			ArgCount:  len(argNames),
			ArgNames:  argNames,
			payloadFn: payloadFn,
		})
	}

	return specs, nil
}

// TODO: finish with subcommands.
func loadSubcommands(csvPath string) (map[string]map[string]string, error) {
	f, err := os.Open(csvPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return readSubcommands(f, defParseFn)
}

type SubCommands map[string]Subcommand

type Subcommand struct {
	Prefix   string
	Cn       byte
	Fn       byte
	Argcount string
}

func readSubcommands(f io.Reader, parseFn parseFunc) (map[string]map[string]string, error) {
	r := csv.NewReader(f)
	r.TrimLeadingSpace = true
	header, err := r.Read()
	if err != nil {
		return nil, err
	}

	result := make(map[string]map[string]string)

	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}

		rowMap := make(map[string]string)
		for i, key := range header {
			rowMap[key] = row[i]
		}

		prefixBytes, err := parseFn(rowMap["prefix"])
		if err != nil || len(prefixBytes) < 2 {
			return nil, fmt.Errorf("invalid subcommand prefix: %v", err)
		}

		// Split into parent command and subcommand part
		for cut := len(prefixBytes) - 1; cut >= 1; cut-- {
			parentKey := hexKey(prefixBytes[:cut])
			subKey := hexKey(prefixBytes[cut:])
			if _, exists := result[parentKey]; !exists {
				result[parentKey] = make(map[string]string)
			}
			result[parentKey][subKey] = rowMap["name"]
			break
		}
	}

	return result, nil
}

func hexKey(b []byte) string {
	var s []string
	for _, bb := range b {
		s = append(s, fmt.Sprintf("%02X", bb))
	}
	return strings.Join(s, " ")
}

func parseHexBytes(s string) ([]byte, error) {
	parts := strings.Fields(s)
	result := make([]byte, len(parts))
	for i, p := range parts {
		b, err := strconv.ParseUint(p, 16, 8)
		if err != nil {
			return nil, fmt.Errorf("invalid hex byte '%s': %w", p, err)
		}
		result[i] = byte(b)
	}
	return result, nil
}

func makePayloadFn(exprStr string, argNames []string) (func([]byte) (int, error), error) {
	exprStr = strings.TrimSpace(exprStr)
	if exprStr == "" {
		return nil, nil
	}

	node, err := parser.ParseExpr(exprStr)
	if err != nil {
		return nil, fmt.Errorf("invalid formula: %w", err)
	}

	return func(args []byte) (int, error) {
		if len(args) != len(argNames) {
			return -1, fmt.Errorf("number of arguments %d != number of argument names %d", len(args), len(argNames))
		}
		env := map[string]int{}
		for i, name := range argNames {
			if i < len(args) {
				env[name] = int(args[i])
			}
		}
		return evalExpr(node, env)
	}, nil
}

func extractIdentifiersFromExpr(expr string) ([]string, error) {
	node, err := parser.ParseExpr(expr)
	if err != nil {
		return nil, err
	}
	return extractIdentifiers(node)
}

// extractIdentifiers extracts identifiers names from the expression.
func extractIdentifiers(node ast.Expr) ([]string, error) {
	seen := map[string]bool{}
	var names []string

	ast.Inspect(node, func(n ast.Node) bool {
		if ident, ok := n.(*ast.Ident); ok {
			if !seen[ident.Name] {
				// You might want to skip "true", "false", etc., if using booleans
				names = append(names, ident.Name)
				seen[ident.Name] = true
			}
		}
		return true
	})
	return names, nil
}

func evalExpr(node ast.Expr, vars map[string]int) (int, error) {
	switch e := node.(type) {
	case *ast.BinaryExpr:
		left, err := evalExpr(e.X, vars)
		if err != nil {
			return 0, err
		}
		right, err := evalExpr(e.Y, vars)
		if err != nil {
			return 0, err
		}
		switch e.Op {
		case token.ADD:
			return left + right, nil
		case token.MUL:
			return left * right, nil
		case token.SUB:
			return left - right, nil
		case token.QUO:
			if right == 0 {
				return 0, fmt.Errorf("division by zero")
			}
			return left / right, nil
		default:
			return 0, fmt.Errorf("unsupported op: %s", e.Op)
		}

	case *ast.Ident:
		val, ok := vars[e.Name]
		if !ok {
			return 0, fmt.Errorf("unknown variable: %s", e.Name)
		}
		return val, nil

	case *ast.BasicLit:
		return strconv.Atoi(e.Value)

	default:
		return 0, fmt.Errorf("unsupported expression: %T", e)
	}
}
