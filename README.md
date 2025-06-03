# Open Send Data Tool for ESC/POS

This is an open-source extended implementation of the EPSON Send Data tool for
ESC/POS printers. It is designed to send raw data to printers using the ESC/POS
command set.

It supports parsing \*.dat files, and a publicly avaiable subset of Senddat
commands found in the documentation:

- `'// ...` - comment
- `*N` - delay N milliseconds
- `.text` - output text and wait for "any key" press.  In this implementation,
  it waits for the user to press Enter.
- `!text` - output text without waiting for a key press.

Senddat commands are read till the end of line. Maximum line length is 250 chars.

## Extensions
In additional to the standard senddat functions, this version is extended to
support the following:

- Go templating language preprocessor (template files up to 1MB)
- TODO: File inclusion via `@file.txt` directive
- TODO: Bitmap data inclusion via `#image.png` (must be monochrome)
- TODO: Binary data inclusion via `&file.bin`

### Templating
Following functions are predefined:

- Range "helpers":
    - `count m n` - returns an iterator that returns all numbers in range $[m..n]$
    - `count_step m n s` - self-explanatory
- String functions:
    - `strcat s1 s2` - concatenates strings "s1" and "s2"
    - `strlen s` - returns a length of a unicode string "s"


## Examples

### Simple example
```plain
'// Initialise printer
    ESC "@"

'// Print Printer intitialised and wait for key press
.Printer initialised, press Enter to continue

`// Prints hello world on the printer.
    "Hello world!" LF

`// Wait for 2 seconds
*2000

`// Print some graphics
    ESC "*" 0 8 0

    0x01 0x02 0x03 0x04 0x05 0x06 0x07 0x08
```

### ESC J interval tester for ESC/POS printer:
```plain
'// Line feed test, output a line in the ESC * mode and see if it connects
    ESC "@"
'// Unidirectional
    ESC "U" 1

'// open source senddat extension, ranges through counter from 15 to 16.
{{ range $n := count 15 16 }}
	"Spacing: {{$n}}" LF
	ESC "*" 0 5 0
	0xFF 0x00 0xFF 0x00 0xFF
	ESC "J" {{ $n }}

	ESC "*" 0 4 0
	0x00 0xFF 0x00 0xFF
	ESC "J" {{ $n }}

	LF
{{end}}

    ESC "@"
    ESC "E" 1 "END OF TEST" LF ESC "E" 0
```

## Motivation
This project was created to provide an open-source alternative to the EPSON
Send Data tool, which is not available for all platforms and has limited
functionality. The goal is to provide a flexible and extensible tool for
sending raw data to ESC/POS printers, with support for templating and other
features that make it easier to work with printers in various environments.

## Installation
```shell
go install github.com/rusq/senddat/cmd/senddat@latest
```

## Legal
Senddat-OS is BSD licensed.

ESC/POS is a registered trademark of Seiko Epson Corporation.

The following examples in testdata/POS are taken from ESC/POS manual, and are (c) Seiko Epson Corp.:
- label.dat
- page_mode.dat
- graphics.dat
- receipt.dat

they are used for unit testing only.
