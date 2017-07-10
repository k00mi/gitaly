package streamio

import (
	"io"
)

// NewReader turns receiver into an io.Reader. Errors from the receiver
// function are passed on unmodified. This means receiver should emit
// io.EOF when done.
func NewReader(receiver func() ([]byte, error)) io.Reader {
	return &receiveReader{receiver: receiver}
}

type receiveReader struct {
	receiver func() ([]byte, error)
	data     []byte
	err      error
}

func (rr *receiveReader) Read(p []byte) (int, error) {
	if len(rr.data) == 0 {
		rr.data, rr.err = rr.receiver()
	}
	n := copy(p, rr.data)
	rr.data = rr.data[n:]
	if len(rr.data) == 0 {
		return n, rr.err
	}
	return n, nil
}

// WriteTo implements io.WriterTo.
func (rr *receiveReader) WriteTo(w io.Writer) (int64, error) {
	var written int64

	// Deal with left-over state in rr.data and rr.err, if any
	if len(rr.data) > 0 {
		n, err := w.Write(rr.data)
		written += int64(n)
		if err != nil {
			return written, err
		}
	}
	if rr.err != nil {
		return written, rr.err
	}

	// Consume the response stream
	var errRead, errWrite error
	var n int
	var buf []byte
	for errWrite == nil && errRead != io.EOF {
		buf, errRead = rr.receiver()
		if errRead != nil && errRead != io.EOF {
			return written, errRead
		}

		if len(buf) > 0 {
			n, errWrite = w.Write(buf)
			written += int64(n)
		}
	}

	return written, errWrite
}

// NewWriter turns sender into an io.Writer.
func NewWriter(sender func(p []byte) error) io.Writer {
	return &sendWriter{sender: sender}
}

var writeBufferSize = 128 * 1024

type sendWriter struct {
	sender func([]byte) error
}

func (sw *sendWriter) Write(p []byte) (int, error) {
	var sent int

	for len(p) > 0 {
		chunkSize := len(p)
		if chunkSize > writeBufferSize {
			chunkSize = writeBufferSize
		}

		if err := sw.sender(p[:chunkSize]); err != nil {
			return sent, err
		}

		sent += chunkSize
		p = p[chunkSize:]
	}

	return sent, nil
}

// ReadFrom implements io.ReaderFrom.
func (sw *sendWriter) ReadFrom(r io.Reader) (int64, error) {
	var nRead int64
	buf := make([]byte, writeBufferSize)

	var errRead, errSend error
	for errSend == nil && errRead != io.EOF {
		var n int

		n, errRead = r.Read(buf)
		nRead += int64(n)
		if errRead != nil && errRead != io.EOF {
			return nRead, errRead
		}

		if n > 0 {
			errSend = sw.sender(buf[:n])
		}
	}

	return nRead, errSend
}
