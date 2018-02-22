package tail

import (
	"bytes"
	"reflect"
	"testing"
)

type config struct {
	lens []int
	n    int
}

func TestWithoutEOL(t *testing.T) {
	t.Parallel()
	l := []int{1, 2, 3, 4}
	testReadLines(t, config{l, 10}, false)
}

func TestReadLongLines(t *testing.T) {
	t.Parallel()
	l := []int{128 * 1024, 256 * 1024, 3, 4, 5}
	testReadLines(t, config{l, 3}, true)
}

// Excerpt from https://github.com/kubernetes/kubernetes/pull/23265/files
func testReadLines(t *testing.T, cfg config, hasEOL bool) {
	var (
		lines    [][]byte
		input    []byte
		expected []byte
	)
	for _, leng := range cfg.lens {
		line := make([]byte, 0, leng)
		for i := 1; i < leng; i++ {
			line = append(line, 'a')
		}
		line = append(line, '\n')
		lines = append(lines, line)
		input = append(input, line...)
	}
	i := len(lines) - cfg.n
	if i < 0 {
		i = 0
	}
	for ; i < len(lines); i++ {
		expected = append(expected, lines[i]...)
	}
	if !hasEOL {
		input = input[:len(input)-1]
	}
	reader := bytes.NewReader(input)
	tail := New(reader, cfg.n)
	buffer := new(bytes.Buffer)
	written, err := tail.WriteTo(buffer)
	if err != nil {
		t.Fatal(err)
	}
	if written != int64(len(expected)) {
		t.Fatalf("expected length: %d, but got %d", len(expected), written)
	}
	if !reflect.DeepEqual(buffer.Bytes(), expected) {
		t.Fatalf("expected content:\n%s\nbut got:\n%s", string(expected), string(buffer.Bytes()))
	}
}
