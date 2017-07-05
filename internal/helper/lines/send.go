package lines

import (
	"bufio"
	"io"
)

// MaxMsgSize establishes the threshold to flush the buffer when using the
// `Send` function. It's a variable instead of a constant to make it easier to
// override in tests.
var MaxMsgSize = 1024 * 128 // 128 KiB

// Sender handles a buffer of lines from a Git command
type Sender func([][]byte) error

type writer struct {
	sender Sender
	size   int
	lines  [][]byte
}

// CopyAndAppend adds a newly allocated copy of `e` to the `s` slice. Useful to
// avoid io buffer shennanigans
func CopyAndAppend(s [][]byte, e []byte) ([][]byte, int) {
	line := make([]byte, len(e))
	size := copy(line, e)
	return append(s, line), size
}

// flush calls the `sender` handler function with the accumulated lines and
// clears the lines buffer.
func (w *writer) flush() error {
	if len(w.lines) == 0 { // No message to send, just return
		return nil
	}

	if err := w.sender(w.lines); err != nil {
		return err
	}

	// Reset the message
	w.lines = nil
	w.size = 0

	return nil
}

// addLine adds a new line to the writer buffer, and flushes if the maximum
// size has been achieved
func (w *writer) addLine(p []byte) error {
	lines, size := CopyAndAppend(w.lines, p)
	w.size += size
	w.lines = lines

	if w.size > MaxMsgSize {
		return w.flush()
	}

	return nil
}

// consume reads from an `io.Reader` and writes each line to the buffer. It
// flushes after being done reading.
func (w *writer) consume(r io.Reader) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		if err := w.addLine(scanner.Bytes()); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return w.flush()
}

// Send reads from `r` and handles the buffered lines using `sender`
func Send(r io.Reader, sender Sender) error {
	writer := &writer{sender: sender}
	return writer.consume(r)
}
