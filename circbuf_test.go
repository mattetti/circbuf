package circbuf_test

import (
	"bytes"
	"io"
	"os"
	"testing"

	mmap "github.com/edsrzf/mmap-go"
	"github.com/mattetti/circbuf"
)

func TestBuffer_Impl(t *testing.T) {
	var _ io.Writer = &circbuf.Buffer{}
}

// it's the caller responsibility to close the file
// and Unmap() the mapped file.
func createTestMmap(t *testing.T, filename string, size int) (*os.File, mmap.MMap) {
	f, err := os.OpenFile(filename+"_testfile", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		f.Close()
		t.Fatal(err)
	}
	if _, err := f.Write(make([]byte, size)); err != nil {
		t.Fatal(err)
	}
	if err := f.Sync(); err != nil {
		t.Fatal(err)
	}
	// map the file
	m, err := mmap.Map(f, mmap.RDWR, 0)
	if err != nil {
		t.Log("something wrong happened when trying to map the file -", err)
		t.Fatal(err)
	}
	return f, m
}

func TestBuffer_ShortWrite(t *testing.T) {
	f, m := createTestMmap(t, t.Name(), 1044)
	defer func() {
		m.Unmap()
		f.Close()
		os.Remove(t.Name() + "_testfile")
	}()

	testCases := []struct {
		name   string
		buffer []byte
	}{
		{name: "memory mapped file", buffer: m},
		{name: "slice of bytes", buffer: make([]byte, 1024)},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			buf, err := circbuf.NewBuffer(tt.buffer, 20, 1024)
			if err != nil {
				t.Fatalf("err: %v", err)
			}

			inp := []byte("hello world")

			n, err := buf.Write(inp)
			if err != nil {
				t.Fatalf("err: %v", err)
			}
			if n != len(inp) {
				t.Fatalf("bad: %v", n)
			}

			if !bytes.Equal(buf.Bytes(), inp) {
				t.Fatalf("bad: %v", buf.Bytes())
			}

		})
	}
}

func TestCircBuffer_FullWrite(t *testing.T) {
	inp := []byte("hello world")

	f, m := createTestMmap(t, t.Name(), len(inp)+20)
	defer func() {
		m.Unmap()
		f.Close()
		os.Remove(t.Name() + "_testfile")
	}()
	buf, err := circbuf.NewBuffer(m, 20, int64(len(inp)))
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	n, err := buf.Write(inp)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if n != len(inp) {
		t.Fatalf("bad: %v", n)
	}

	if !bytes.Equal(buf.Bytes(), inp) {
		t.Fatalf("bad: %v", buf.Bytes())
	}
}

func TestCircBuffer_LongWrite(t *testing.T) {
	inp := []byte("hello world")

	f, m := createTestMmap(t, t.Name(), 6+12)
	defer func() {
		m.Unmap()
		f.Close()
		os.Remove(t.Name() + "_testfile")
	}()
	buf, err := circbuf.NewBuffer(m, 12, 6)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	n, err := buf.Write(inp)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if n != len(inp) {
		t.Fatalf("bad: %v", n)
	}

	expect := []byte(" world")
	if !bytes.Equal(buf.Bytes(), expect) {
		t.Fatalf("bad: %s", buf.Bytes())
	}
}

func TestCircBuffer_HugeWrite(t *testing.T) {
	inp := []byte("hello world")

	f, m := createTestMmap(t, t.Name(), 3+12)
	defer func() {
		m.Unmap()
		f.Close()
		os.Remove(t.Name() + "_testfile")
	}()
	buf, err := circbuf.NewBuffer(m, 12, 3)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	n, err := buf.Write(inp)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if n != len(inp) {
		t.Fatalf("bad: %v", n)
	}

	expect := []byte("rld")
	if !bytes.Equal(buf.Bytes(), expect) {
		t.Fatalf("bad: %s", buf.Bytes())
	}
}

func TestCircBuffer_ManySmall(t *testing.T) {
	inp := []byte("hello world")

	f, m := createTestMmap(t, t.Name(), 3+12)
	defer func() {
		m.Unmap()
		f.Close()
		os.Remove(t.Name() + "_testfile")
	}()
	header := []byte{'m', 'a', 't', 't'}
	for i, r := range header {
		m[i] = r
	}

	buf, err := circbuf.NewBuffer(m, 12, 3)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	for _, b := range inp {
		n, err := buf.Write([]byte{b})
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if n != 1 {
			t.Fatalf("bad: %v", n)
		}
	}

	expect := []byte("rld")
	if !bytes.Equal(buf.Bytes(), expect) {
		t.Fatalf("bad: %v", buf.Bytes())
	}

	for i, r := range header {
		if m[i] != r {
			t.Fatalf("expected %v but got %v\n", r, m[i])
		}
	}
}

func TestCircBuffer_MultiPart(t *testing.T) {
	inputs := [][]byte{
		[]byte("hello world\n"),
		[]byte("this is a test\n"),
		[]byte("my cool input\n"),
	}
	total := 0

	f, m := createTestMmap(t, t.Name(), 4+16)
	defer func() {
		m.Unmap()
		f.Close()
		os.Remove(t.Name() + "_testfile")
	}()
	buf, err := circbuf.NewBuffer(m, 4, 16)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	for _, b := range inputs {
		total += len(b)
		n, err := buf.Write(b)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if n != len(b) {
			t.Fatalf("bad: %v", n)
		}
	}

	if int64(total) != buf.TotalWritten() {
		t.Fatalf("bad total")
	}

	expect := []byte("t\nmy cool input\n")
	if !bytes.Equal(buf.Bytes(), expect) {
		t.Logf("expected: `%s`, got: `%s`\n", string(expect), string(buf.Bytes()))
		t.Fatalf("bad: %v", buf.Bytes())
	}

}

func TestCircBuffer_Reset(t *testing.T) {
	// Write a bunch of data
	inputs := [][]byte{
		[]byte("hello world\n"),
		[]byte("this is a test\n"),
		[]byte("my cool input\n"),
	}

	f, m := createTestMmap(t, t.Name(), 4+4)
	defer func() {
		m.Unmap()
		f.Close()
		os.Remove(t.Name() + "_testfile")
	}()
	buf, err := circbuf.NewBuffer(m, 4, 4)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	for _, b := range inputs {
		n, err := buf.Write(b)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if n != len(b) {
			t.Fatalf("bad: %v", n)
		}
	}

	// Reset it
	buf.Reset()

	// Write more data
	input := []byte("hello")
	n, err := buf.Write(input)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if n != len(input) {
		t.Fatalf("bad: %v", n)
	}

	// Test the output
	expect := []byte("ello")
	if !bytes.Equal(buf.Bytes(), expect) {
		t.Fatalf("bad: %v", string(buf.Bytes()))
	}
}
