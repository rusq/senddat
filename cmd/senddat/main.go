package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/rusq/senddat"
)

var params = struct {
	input      string
	output     string
	isTemplate bool
	verbose    bool
	reverse    bool
}{
	output: "",
	input:  "",
}

func init() {
	flag.Usage = usage
	flag.BoolVar(&params.isTemplate, "t", false, "treat input as a Go template")
	flag.StringVar(&params.output, "o", "", "output file (default stdout)")
	flag.BoolVar(&params.verbose, "v", os.Getenv("DEBUG") == "1", "enable verbose logging")
	flag.BoolVar(&params.reverse, "r", false, "reverse the PRN file")
}

func main() {
	flag.Parse()

	if params.verbose {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	if flag.NArg() > 0 {
		params.input = flag.Arg(0)
	}

	ctx := context.Background()

	parseFn := senddat.Parse
	if params.isTemplate {
		parseFn = senddat.ParseFromTemplate
	}

	var err error
	if params.reverse {
		err = reverse(ctx, params.input, params.output)
	} else {
		err = parse(ctx, params.input, params.output, parseFn)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func parse(_ context.Context, input string, output string, parseFn func(w io.Writer, r io.Reader) error) error {
	r, w, err := openFiles(input, output)
	if err != nil {
		return fmt.Errorf("failed to open files: %w", err)
	}
	defer r.Close()
	defer w.Close()

	if err := parseFn(w, r); err != nil {
		return fmt.Errorf("failed to parse input: %w", err)
	}
	slog.Info("Data sent successfully", "output", output, "input", input)
	return nil
}

func stringRenderFn(w io.Writer) func(senddat.Entry) error {
	return func(entry senddat.Entry) error {
		_, err := fmt.Fprintln(w, entry.String())
		return err
	}
}

func reverse(_ context.Context, input string, output string) error {
	r, w, err := openFiles(input, output)
	if err != nil {
		return fmt.Errorf("failed to open files: %w", err)
	}
	defer r.Close()
	defer w.Close()
	renderFn := stringRenderFn(w)

	entries, err := senddat.Decode(r, senddat.GenericCommandSpecs)
	if err != nil {
		return fmt.Errorf("failed to decode input: %w", err)
	}

	for _, entry := range entries {
		if err := renderFn(entry); err != nil {
			return fmt.Errorf("failed to render entry: %w", err)
		}
	}

	slog.Info("Data reversed successfully", "output", output, "input", input)
	return nil
}

func usage() {
	out := flag.CommandLine.Output()
	fmt.Fprintf(out, "Open Send Data Tool - parses ESC/POS dat files and sends data to a file or a printer.\n")
	fmt.Fprintf(out, "It does the same as Epson ESC/POS senddat [1] utility, but can be compiled for\n")
	fmt.Fprintf(out, "different platforms and architectures.")
	fmt.Fprintf(out, "\t[1]: https://download.ebz.epson.net/dsc/du/02/DriverDownloadInfo.do?LG2=EN&CN2=US&CTI=381&PRN=TM-m30II&OSC=W1164\n\n")
	fmt.Fprintf(out, "Usage: %s [-o <output>] [input]\n\n", os.Args[0])
	fmt.Fprintf(out, "Flags:\n")
	flag.PrintDefaults()
}

func openFiles(input string, output string) (io.ReadCloser, io.WriteCloser, error) {
	var r io.ReadCloser
	if input == "" || input == "-" {
		r = os.Stdin
	} else {
		file, err := os.Open(input)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to open input file: %w", err)
		}
		r = file
	}

	var w io.WriteCloser
	if output == "" || output == "-" {
		w = os.Stdout
	} else {
		file, err := os.Create(output)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create output file: %w", err)
		}
		w = file
	}
	return r, w, nil
}
