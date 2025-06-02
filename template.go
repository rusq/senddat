package senddat

import (
	"bytes"
	"io"
	"iter"
	"text/template"
	"unicode/utf8"
)

var tmplfuncs = template.FuncMap{
	"count":      count,
	"count_step": countStep,
	"strlen":     utf8.RuneCountInString,
	"strcat":     strcat,
}

var basetmpl = template.New("").Funcs(tmplfuncs)

func ParseFromTemplate(w io.Writer, r io.Reader) error {
	data, err := io.ReadAll(io.LimitReader(r, 1048576))
	if err != nil {
		return err
	}
	tmpl, err := basetmpl.Parse(string(data))
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, nil); err != nil {
		return err
	}
	return Parse(w, &buf)
}

// count returns an iterator that will count from [start..end]
func count(start, end int) iter.Seq[int] {
	return countStep(start, end, 1)
}

func countStep(start, end, step int) iter.Seq[int] {
	return func(yield func(int) bool) {
		for i := start; i <= end; i += step {
			if !yield(i) {
				return
			}
		}
	}
}

func strcat(s1, s2 string) string {
	return s1 + s2
}
