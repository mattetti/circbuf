package circbuf

import "fmt"

// Buffer implements a circular buffer. It is a fixed size,
// and new writes overwrite older data, such that for a buffer
// of size N, for any amount of writes, only the last N bytes
// are retained.
type Buffer struct {
	data []byte
	// size doesn't include an optional offset
	size        int64
	writeCursor int64
	readCursor  int64
	written     int64
	offset      int64
}

// NewBuffer sets a new circular buffer on top of the passed slice of bytes.
// A certain amount of bytes can be skipped if used as flags for instance and
// the length of the buffer must also be set.
func NewBuffer(m []byte, skip, size int64) (*Buffer, error) {
	b := &Buffer{
		offset: skip,
		size:   size,
		data:   m,
	}
	return b, nil
}

// Write writes up to len(buf) bytes to the internal ring,
// overriding older data if necessary.
func (b *Buffer) Write(buf []byte) (int, error) {
	// Account for total bytes written
	n := len(buf)
	b.written += int64(n)

	// If the buffer is larger than ours, then we only care
	// about the last size bytes anyways
	if int64(n) > b.size {
		buf = buf[int64(n)-b.size:]
	}

	// Copy in place
	remain := b.size - b.writeCursor
	copy(b.data[b.offset+b.writeCursor:], buf)
	if int64(len(buf)) > remain {
		copy(b.data[b.offset:], buf[remain:])
	}

	// Update location of the cursor
	b.writeCursor = ((b.writeCursor + int64(len(buf))) % b.size)
	return n, nil
}

// Size returns the size of the buffer
func (b *Buffer) Size() int64 {
	return b.size
}

// TotalWritten provides the total number of bytes written
func (b *Buffer) TotalWritten() int64 {
	return b.written
}

// Read reads up to len(p) bytes into p. It returns the number of bytes read (0
// <= n <= len(p)) and any error encountered. Even if Read returns n < len(p),
// it may use all of p as scratch space during the call. If some data is
// available but not len(p) bytes, Read conventionally returns what is available
// instead of waiting for more.
func (b *Buffer) Read(out []byte) (n int, err error) {
	if b.readCursor >= b.Size() {
		// we read the entire buffer, let's loop back to the beginning
		b.readCursor = 0
	} else if b.readCursor+int64(len(out)) > b.Size() {
		// we don't have enough data in our buffer to fill the passed buffer
		// we need to do multiple passes
		n := copy(out, b.data[b.offset+b.readCursor:])
		b.readCursor += int64(n)
		// TMP check, should remove
		if b.readCursor != b.Size() {
			panic(fmt.Sprintf("off by one much? %d - %d", b.readCursor, b.Size()))
		}
		n2, _ := b.Read(out[n:])
		b.readCursor += int64(n2)
		return int(n + n2), nil
	}
	n = copy(out, b.data[b.offset+b.readCursor:])
	return
}

// Bytes provides a slice of the bytes written. This
// slice should not be written to.
func (b *Buffer) Bytes() []byte {
	switch {
	case b.written >= b.size && b.writeCursor == 0:
		return b.data[b.offset:]
	case b.written > b.size:
		out := make([]byte, b.size)
		copy(out,
			b.data[b.offset+b.writeCursor:])
		copy(out[b.size-b.writeCursor:],
			b.data[b.offset:b.offset+b.writeCursor])
		return out
	default:
		return b.data[b.offset : b.offset+b.writeCursor]
	}
}

// Reset resets the buffer so it has no content.
func (b *Buffer) Reset() {
	b.writeCursor = 0
	b.written = 0
}
