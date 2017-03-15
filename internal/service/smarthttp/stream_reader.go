package smarthttp

type bytesReceiver interface {
	ReceiveBytes() ([]byte, error)
}

type streamReader struct {
	br  bytesReceiver
	buf []byte
}

func (rd *streamReader) Read(p []byte) (int, error) {
	var err error

	if len(rd.buf) == 0 {
		rd.buf, err = rd.br.ReceiveBytes()
		if err != nil {
			return 0, err
		}
	}
	n := copy(p, rd.buf)
	rd.buf = rd.buf[n:]
	return n, nil
}
