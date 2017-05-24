package circbuf_test

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"testing"

	mmap "github.com/edsrzf/mmap-go"
	"github.com/mattetti/circbuf"
)

func ExampleNewBuffer() {
	myBuf := make([]byte, 6)
	buf, _ := circbuf.NewBuffer(myBuf, 0, int64(len(myBuf)))
	buf.Write([]byte("hello world"))

	fmt.Println(string(buf.Bytes()))
	// Output: world
}

func ExampleWrite() {
	// this example shows how to use a memory mapped file
	f, err := os.OpenFile("exampleTestfile", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		panic(err)
	}
	defer func() {
		f.Close()
		os.Remove("exampleTestfile")
	}()
	if _, err := f.Write(make([]byte, 2+7)); err != nil {
		panic(err)
	}
	if err := f.Sync(); err != nil {
		panic(err)
	}
	// map the file
	m, err := mmap.Map(f, mmap.RDWR, 0)
	if err != nil {
		panic(err)
	}
	defer m.Unmap()
	buf, _ := circbuf.NewBuffer(m, 2, 7)
	buf.Write([]byte("hello world, I am a circular buffer!"))

	fmt.Println(string(buf.Bytes()))
	// Output: buffer!
}

func TestBuffer_Impl(t *testing.T) {
	var _ io.Writer = &circbuf.Buffer{}
	var _ io.Reader = &circbuf.Buffer{}
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

func TestBuffer_ShortRead(t *testing.T) {
	f, m := createTestMmap(t, t.Name(), 4+11)
	defer func() {
		m.Unmap()
		f.Close()
		os.Remove(t.Name() + "_testfile")
	}()

	testCases := []struct {
		name   string
		buffer []byte
		size   int
		offset int
	}{
		{name: "memory mapped file", size: 11, offset: 4, buffer: m},
		{name: "slice of bytes", size: 11, offset: 4, buffer: make([]byte, 4+11)},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			buf, err := circbuf.NewBuffer(tt.buffer, int64(tt.offset), int64(tt.size))
			if err != nil {
				t.Fatalf("err: %v", err)
			}

			inp := []byte("hello world")

			n, err := buf.Write(inp)
			if err != nil {
				t.Fatalf("err: %v", err)
			}

			out := make([]byte, len(inp))
			n, _ = buf.Read(out)
			if n != len(inp) {
				t.Fatalf("expected to read %d bytes, but read %d", len(inp), n)
			}
			if bytes.Compare(inp, out) != 0 {
				t.Fatalf("expected to read the same data as what was written but got %q instead of %q", out, inp)
			}

			t.Run("read in a loop", func(t *testing.T) {
				out = make([]byte, 2*tt.size)
				n, _ = buf.Read(out)
				if n != len(out) {
					t.Fatalf("expected to read 2*%d bytes, but read %d", len(out), n)
				}
				expected := append(inp, inp...)
				if bytes.Compare(expected, out) != 0 {
					t.Fatalf("expected the content of the buffer to be %q but was %q", expected, out)
				}
			})

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
