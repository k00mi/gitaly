package git2go

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"path"
	"strings"

	"gitlab.com/gitlab-org/gitaly/internal/command"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/config"
)

func run(ctx context.Context, cfg config.Cfg, subcommand string, arg string) (string, error) {
	binary := path.Join(cfg.BinDir, "gitaly-git2go")

	var stderr, stdout bytes.Buffer
	cmd, err := command.New(ctx, exec.Command(binary, subcommand, "-request", arg), nil, &stdout, &stderr)
	if err != nil {
		return "", err
	}

	if err := cmd.Wait(); err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("%s", stderr.String())
		}
		return "", err
	}

	return stdout.String(), nil
}

func serialize(v interface{}) (string, error) {
	marshalled, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(marshalled), nil
}

func deserialize(serialized string, v interface{}) error {
	base64Decoder := base64.NewDecoder(base64.StdEncoding, strings.NewReader(serialized))
	jsonDecoder := json.NewDecoder(base64Decoder)
	return jsonDecoder.Decode(v)
}

func serializeTo(writer io.Writer, v interface{}) error {
	base64Encoder := base64.NewEncoder(base64.StdEncoding, writer)
	defer base64Encoder.Close()
	jsonEncoder := json.NewEncoder(base64Encoder)
	return jsonEncoder.Encode(v)
}
