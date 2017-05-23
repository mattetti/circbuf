package circbuf

// Buffer implements a circular buffer. It is a fixed size,
// and new writes overwrite older data, such that for a buffer
// of size N, for any amount of writes, only the last N bytes
// are retained.
type Buffer struct {
	data []byte
	// size doesn't include an optional offset
	size        int64
	writeCursor int64
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
