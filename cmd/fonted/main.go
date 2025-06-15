// Command fonted allows you to view and edit font files to load into
// the printer.
package main

import (
	"errors"
	"flag"
	"io"
	"os"
)

var (
	prn = flag.Bool("p", false, "print the font to the console")
)

func main() {
	printString(os.Stdout, 9, []byte("786"), [2]rune{' ', 'X'})
}

func printString(w io.Writer, pins int, str []byte, disp [2]rune) error {
	if pins != 9 && pins != 24 {
		return errors.New("unsupported pin count")
	}
	for _, char := range str {
		offset := int(char - 32)
		if offset < 0 || offset >= len(RobotronFont)/pins {
			return errors.New("character out of range")
		}
		if err := printChar(w, pins, RobotronFont[offset*pins:offset*pins+pins], disp); err != nil {
			return err
		}
	}
	return nil
}

func printChar(w io.Writer, pins int, character []uint16, disp [2]rune) error {
	if pins != 9 && pins != 24 {
		return errors.New("unsupported pin count")
	}
	for _, b := range character {
		for i := range pins {
			if b&(1<<i) != 0 {
				_, err := w.Write([]byte(string(disp[1])))
				if err != nil {
					return err
				}
			} else {
				_, err := w.Write([]byte(string(disp[0])))
				if err != nil {
					return err
				}
			}
		}
		if _, err := w.Write([]byte("\n")); err != nil {
			return err
		}
	}
	return nil
}
