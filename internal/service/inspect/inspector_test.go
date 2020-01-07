package inspect

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus/ctxlogrus"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestWrite(t *testing.T) {
	dest := &bytes.Buffer{}
	writer := Write(dest, func(reader io.Reader) {
		data, err := ioutil.ReadAll(reader)
		require.NoError(t, err)
		require.Equal(t, data, "test\x02data")
	})

	_, err := io.Copy(writer, strings.NewReader("test\x02data"))
	require.NoError(t, err)
}

func TestLogPackInfoStatistic(t *testing.T) {
	dest := &bytes.Buffer{}
	log := &logrus.Logger{
		Out:       dest,
		Formatter: new(logrus.JSONFormatter),
		Level:     logrus.InfoLevel,
	}
	ctx := ctxlogrus.ToContext(context.Background(), log.WithField("test", "logging"))

	logging := LogPackInfoStatistic(ctx)
	logging(strings.NewReader("0038\x41ACK 1e292f8fedd741b75372e19097c76d327140c312 ready\n0035\x02Total 1044 (delta 519), reused 1035 (delta 512)\n0038\x41ACK 1e292f8fedd741b75372e19097c76d327140c312 ready\n0000\x01"))

	require.Contains(t, dest.String(), "Total 1044 (delta 519), reused 1035 (delta 512)")
	require.NotContains(t, dest.String(), "ACK 1e292f8fedd741b75372e19097c76d327140c312")
}
