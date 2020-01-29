package tabwriter

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

const (
	minWidth = 6
	width    = 4
	padding  = 3
	padChar  = ' '
)

type Writer struct {
	delegate *tabwriter.Writer

	header []string
	buf    bytes.Buffer
}

func toStringList(args ...interface{}) []string {
	strLst := make([]string, 0, len(args))
	for _, arg := range args {
		strLst = append(strLst, fmt.Sprint(arg))
	}
	return strLst
}

func (w *Writer) Render() error {
	// print header
	_, err := fmt.Fprintln(w.delegate, strings.Join(w.header, "\t"))
	if err != nil {
		return err
	}

	// print content
	_, err = w.buf.WriteTo(w.delegate)
	if err != nil {
		return err
	}
	return w.delegate.Flush()
}

func (w *Writer) Append(args ...interface{}) {
	_, _ = fmt.Fprintln(&w.buf, strings.Join(toStringList(args...), "\t"))
}

func (w *Writer) SetHeader(header []string) {
	upperHeader := make([]string, 0, len(header))
	for _, col := range header {
		upperHeader = append(upperHeader, strings.ToUpper(col))
	}
	w.header = upperHeader
}

func New(out io.Writer) *Writer {
	return &Writer{
		delegate: tabwriter.NewWriter(out,
			minWidth,
			width,
			padding,
			padChar,
			0),
	}
}
