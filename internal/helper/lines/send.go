package lines

import (
	"bufio"
	"bytes"
	"io"
)

type SenderOpts struct {
	Delimiter []byte
}

// ItemsPerMessage establishes the threshold to flush the buffer when using the
// `Send` function. It's a variable instead of a constant to make it possible to
// override in tests.
var ItemsPerMessage = 20

// Sender handles a buffer of lines from a Git command
type Sender func([][]byte) error

type writer struct {
	sender  Sender
	lines   [][]byte
	options SenderOpts
}

// CopyAndAppend adds a newly allocated copy of `e` to the `s` slice. Useful to
// avoid io buffer shennanigans
func CopyAndAppend(s [][]byte, e []byte) [][]byte {
	line := make([]byte, len(e))
	copy(line, e)
	return append(s, line)
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

	return nil
}

// addLine adds a new line to the writer buffer, and flushes if the maximum
// size has been achieved
func (w *writer) addLine(p []byte) error {
	w.lines = CopyAndAppend(w.lines, p)

	if len(w.lines) >= ItemsPerMessage {
		return w.flush()
	}

	return nil
}

// consume reads from an `io.Reader` and writes each line to the buffer. It
// flushes after being done reading.
func (w *writer) consume(r io.Reader) error {
	buf := bufio.NewReader(r)

	for finished := false; !finished; {
		var line []byte

		for {
			// delim can be multiple bytes, so we read till the end byte of it ...
			chunk, err := buf.ReadBytes(w.delimiter()[len(w.delimiter())-1])
			if err != nil && err != io.EOF {
				return err
			}

			line = append(line, chunk...)
			// ... then we check if the last bytes of line are the same as delim
			if bytes.HasSuffix(line, w.delimiter()) {
				break
			}

			if err == io.EOF {
				finished = true
				break
			}
		}

		line = bytes.TrimRight(line, string(w.delimiter()))
		if len(line) == 0 {
			break
		}

		if err := w.addLine(line); err != nil {
			return err
		}
	}

	return w.flush()
}

func (w *writer) delimiter() []byte { return w.options.Delimiter }

// Send reads output from `r`, splits it at `opts.Delimiter``, then handles the
// buffered lines using `sender`.
func Send(r io.Reader, sender Sender, opts SenderOpts) error {
	if len(opts.Delimiter) == 0 {
		opts.Delimiter = []byte{'\n'}
	}

	writer := &writer{sender: sender, options: opts}
	return writer.consume(r)
}
