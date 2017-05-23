circbuf
=======

This repository provides the `circbuf` package. This provides a `Buffer` object
which is a circular (or ring) buffer. It has a fixed size, but can be written
to infinitely. Only the last `size` bytes are ever retained. The buffer implements
the `io.Writer` interface.
This is a fork of Armon's implementation https://github.com/armon/circbuf but
adapted so it can be backed by a memory mapped file or a pre allocated buffer.
The main change comes from the fact that the backing buffer is provided and an
offset can be used. The content of the offset is not overwritten and can be used
to write/read flags for instance.

Documentation
=============

Full documentation can be found on [Godoc](http://godoc.org/github.com/mattetti/circbuf)

Usage
=====

The `circbuf` package ca be used with a pre allocated buffer:

```go
myBuf := make([]byte, 6)
buf, _ := circbuf.NewBuffer(myBuf, 0, len(myBuf))
buf.Write([]byte("hello world"))

if string(buf.Bytes()) != " world" {
    panic("should only have last 6 bytes!")
}

```

Or with a memory mapped file:

