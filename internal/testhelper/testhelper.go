package testhelper

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus/ctxlogrus"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/gitaly/internal/command"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/config"
	"gitlab.com/gitlab-org/gitaly/internal/helper/text"
	"gitlab.com/gitlab-org/gitaly/internal/metadata/featureflag"
	"gitlab.com/gitlab-org/gitaly/internal/storage"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
	"google.golang.org/grpc/metadata"
)

const (
	RepositoryAuthToken = "the-secret-token"
	DefaultStorageName  = "default"
	GlID                = "user-123"
)

var (
	TestUser = &gitalypb.User{
		Name:       []byte("Jane Doe"),
		Email:      []byte("janedoe@gitlab.com"),
		GlId:       GlID,
		GlUsername: "janedoe",
	}
)

// MustReadFile returns the content of a file or fails at once.
func MustReadFile(t testing.TB, filename string) []byte {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}

	return content
}

// GitlabTestStoragePath returns the storage path to the gitlab-test repo.
func GitlabTestStoragePath() string {
	if testDirectory == "" {
		log.Fatal("you must call testhelper.Configure() before GitlabTestStoragePath()")
	}
	return filepath.Join(testDirectory, "storage")
}

// GitalyServersMetadata returns a metadata pair for gitaly-servers to be used in
// inter-gitaly operations.
func GitalyServersMetadata(t testing.TB, serverSocketPath string) metadata.MD {
	gitalyServers := storage.GitalyServers{
		"default": storage.ServerInfo{
			Address: serverSocketPath,
			Token:   RepositoryAuthToken,
		},
	}

	gitalyServersJSON, err := json.Marshal(gitalyServers)
	if err != nil {
		t.Fatal(err)
	}

	return metadata.Pairs("gitaly-servers", base64.StdEncoding.EncodeToString(gitalyServersJSON))
}

// MustRunCommand runs a command with an optional standard input and returns the standard output, or fails.
func MustRunCommand(t testing.TB, stdin io.Reader, name string, args ...string) []byte {
	if t != nil {
		t.Helper()
	}

	var cmd *exec.Cmd
	if name == "git" {
		cmd = exec.Command(config.Config.Git.BinPath, args...)
		cmd.Env = os.Environ()
		cmd.Env = append(command.GitEnv, cmd.Env...)
		cmd.Env = append(cmd.Env,
			"GIT_AUTHOR_DATE=1572776879 +0100",
			"GIT_COMMITTER_DATE=1572776879 +0100",
		)
	} else {
		cmd = exec.Command(name, args...)
	}

	if stdin != nil {
		cmd.Stdin = stdin
	}

	output, err := cmd.Output()
	if err != nil {
		stderr := err.(*exec.ExitError).Stderr
		if t == nil {
			log.Print(name, args)
			log.Printf("%s", stderr)
			log.Fatal(err)
		} else {
			t.Log(name, args)
			t.Logf("%s", stderr)
			t.Fatal(err)
		}
	}

	return output
}

// GetTemporaryGitalySocketFileName will return a unique, useable socket file name
func GetTemporaryGitalySocketFileName() string {
	if testDirectory == "" {
		log.Fatal("you must call testhelper.Configure() before GetTemporaryGitalySocketFileName()")
	}

	tmpfile, err := ioutil.TempFile(testDirectory, "gitaly.socket.")
	if err != nil {
		// No point in handling this error, panic
		panic(err)
	}

	name := tmpfile.Name()
	tmpfile.Close()
	os.Remove(name)

	return name
}

// GetLocalhostListener listens on the next available TCP port and returns
// the listener and the localhost address (host:port) string.
func GetLocalhostListener(t testing.TB) (net.Listener, string) {
	l, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)

	addr := fmt.Sprintf("localhost:%d", l.Addr().(*net.TCPAddr).Port)

	return l, addr
}

// MustHaveNoChildProcess panics if it finds a running or finished child
// process. It waits for 2 seconds for processes to be cleaned up by other
// goroutines.
func MustHaveNoChildProcess() {
	waitDone := make(chan struct{})
	go func() {
		command.WaitAllDone()
		close(waitDone)
	}()

	select {
	case <-waitDone:
	case <-time.After(2 * time.Second):
	}

	mustFindNoFinishedChildProcess()
	mustFindNoRunningChildProcess()
}

func mustFindNoFinishedChildProcess() {
	// Wait4(pid int, wstatus *WaitStatus, options int, rusage *Rusage) (wpid int, err error)
	//
	// We use pid -1 to wait for any child. We don't care about wstatus or
	// rusage. Use WNOHANG to return immediately if there is no child waiting
	// to be reaped.
	wpid, err := syscall.Wait4(-1, nil, syscall.WNOHANG, nil)
	if err == nil && wpid > 0 {
		panic(fmt.Errorf("wait4 found child process %d", wpid))
	}
}

func mustFindNoRunningChildProcess() {
	pgrep := exec.Command("pgrep", "-P", fmt.Sprintf("%d", os.Getpid()))
	desc := fmt.Sprintf("%q", strings.Join(pgrep.Args, " "))

	out, err := pgrep.Output()
	if err == nil {
		pidsComma := strings.Replace(text.ChompBytes(out), "\n", ",", -1)
		psOut, _ := exec.Command("ps", "-o", "pid,args", "-p", pidsComma).Output()
		panic(fmt.Errorf("found running child processes %s:\n%s", pidsComma, psOut))
	}

	if status, ok := command.ExitStatus(err); ok && status == 1 {
		// Exit status 1 means no processes were found
		return
	}

	panic(fmt.Errorf("%s: %v", desc, err))
}

// ContextOpt returns a new context instance with the new additions to it.
type ContextOpt func(context.Context) (context.Context, func())

// ContextWithTimeout allows to set provided timeout to the context.
func ContextWithTimeout(duration time.Duration) ContextOpt {
	return func(ctx context.Context) (context.Context, func()) {
		return context.WithTimeout(ctx, duration)
	}
}

// ContextWithLogger allows to inject provided logger into the context.
func ContextWithLogger(logger *log.Entry) ContextOpt {
	return func(ctx context.Context) (context.Context, func()) {
		return ctxlogrus.ToContext(ctx, logger), func() {}
	}
}

// Context returns a cancellable context.
func Context(opts ...ContextOpt) (context.Context, func()) {
	ctx, cancel := context.WithCancel(context.Background())
	for _, ff := range featureflag.All {
		ctx = featureflag.IncomingCtxWithFeatureFlag(ctx, ff)
		ctx = featureflag.OutgoingCtxWithFeatureFlags(ctx, ff)
	}

	cancels := make([]func(), len(opts)+1)
	cancels[0] = cancel
	for i, opt := range opts {
		ctx, cancel = opt(ctx)
		cancels[i+1] = cancel
	}

	return ctx, func() {
		for i := len(cancels) - 1; i >= 0; i-- {
			cancels[i]()
		}
	}
}

// GitalySSHParams contains parameters used to exec 'gitaly-ssh' binary.
type GitalySSHParams struct {
	Args    []string
	EnvVars []string
}

// ListenGitalySSHCalls creates a script that intercepts 'gitaly-ssh' binary calls.
// It substitutes execution path of 'gitaly-ssh' with a path to a script and returns a modified configuration to be used.
// The second return parameter provides the list of parameters used in each invocation of the 'gitaly-ssh'.
func ListenGitalySSHCalls(t *testing.T, conf config.Cfg) (config.Cfg, func() []GitalySSHParams, Cleanup) {
	t.Helper()

	if conf.BinDir == "" {
		assert.FailNow(t, "BinDir must be set")
		return conf, func() []GitalySSHParams { return nil }, func() {}
	}

	const envPrefix = "env-"
	const argsPrefix = "args-"

	tmpDir, clean := TempDir(t)
	script := fmt.Sprintf(`
		#!/bin/sh

		# To omit possible problem with parallel run and a race for the file creation with '>'
		# this option is used, please checkout https://mywiki.wooledge.org/NoClobber for more details.
		set -o noclobber

		ENV_IDX=$(ls %[1]q | grep %[2]s | wc -l)
		env > "%[1]s/%[2]s$ENV_IDX"

		ARGS_IDX=$(ls %[1]q | grep %[3]s | wc -l)
		echo $@ > "%[1]s/%[3]s$ARGS_IDX"

		%[4]q "$@" 1>&1 2>&2
		exit $?`,
		tmpDir, envPrefix, argsPrefix, filepath.Join(conf.BinDir, "gitaly-ssh"))

	require.NoError(t, ioutil.WriteFile(filepath.Join(tmpDir, "gitaly-ssh"), []byte(script), 0755))
	conf.BinDir = tmpDir

	getSSHParams := func() []GitalySSHParams {
		var gitalySSHParams []GitalySSHParams
		err := filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			filename := filepath.Base(path)

			parseParams := func(prefix, delim string) error {
				if !strings.HasPrefix(filename, prefix) {
					return nil
				}

				idx, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(filename, prefix)))
				if err != nil {
					return err
				}

				if len(gitalySSHParams) < idx+1 {
					tmp := make([]GitalySSHParams, idx+1)
					copy(tmp, gitalySSHParams)
					gitalySSHParams = tmp
				}

				data, err := ioutil.ReadFile(path)
				if err != nil {
					return err
				}

				params := strings.Split(string(data), delim)

				switch prefix {
				case argsPrefix:
					gitalySSHParams[idx].Args = params
				case envPrefix:
					gitalySSHParams[idx].EnvVars = params
				}

				return nil
			}

			if err := parseParams(envPrefix, "\n"); err != nil {
				return err
			}

			if err := parseParams(argsPrefix, " "); err != nil {
				return err
			}

			return nil
		})
		assert.NoError(t, err)
		return gitalySSHParams
	}

	return conf, getSSHParams, clean
}

// AssertPathNotExists asserts true if the path doesn't exist, false otherwise
func AssertPathNotExists(t testing.TB, path string) {
	_, err := os.Stat(path)
	assert.True(t, os.IsNotExist(err), "file should not exist: %s", path)
}

// TempDir is a wrapper around ioutil.TempDir that provides a cleanup function.
func TempDir(t testing.TB) (string, func()) {
	if testDirectory == "" {
		log.Fatal("you must call testhelper.Configure() before TempDir()")
	}

	tmpDir, err := ioutil.TempDir(testDirectory, "")
	require.NoError(t, err)

	return tmpDir, func() { require.NoError(t, os.RemoveAll(tmpDir)) }
}

// Cleanup functions should be called in a defer statement
// immediately after they are returned from a test helper
type Cleanup func()

// WriteExecutable ensures that the parent directory exists, and writes an executable with provided content
func WriteExecutable(path string, content []byte) (func(), error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return func() {}, err
	}

	return func() {
		os.RemoveAll(dir)
	}, ioutil.WriteFile(path, content, 0755)
}

// ModifyEnvironment will change an environment variable and return a func suitable
// for `defer` to change the value back.
func ModifyEnvironment(t testing.TB, key string, value string) func() {
	t.Helper()

	oldValue, hasOldValue := os.LookupEnv(key)
	require.NoError(t, os.Setenv(key, value))
	return func() {
		if hasOldValue {
			require.NoError(t, os.Setenv(key, oldValue))
		} else {
			require.NoError(t, os.Unsetenv(key))
		}
	}
}

// GenerateTestCerts creates a certificate that can be used to establish TLS protected TCP connection.
// It returns paths to the file with the certificate and its private key.
func GenerateTestCerts(t *testing.T) (string, string, Cleanup) {
	t.Helper()

	rootCA := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(0, 0, 1),
		BasicConstraintsValid: true,
		IsCA:                  true,
		IPAddresses:           []net.IP{net.ParseIP("0.0.0.0"), net.ParseIP("127.0.0.1"), net.ParseIP("::1"), net.ParseIP("::")},
		DNSNames:              []string{"localhost"},
		KeyUsage:              x509.KeyUsageCertSign,
	}

	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	caCert, err := x509.CreateCertificate(rand.Reader, rootCA, rootCA, &caKey.PublicKey, caKey)
	require.NoError(t, err)

	entityKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	entityX509 := &x509.Certificate{
		SerialNumber: big.NewInt(2),
	}

	entityCert, err := x509.CreateCertificate(rand.Reader, rootCA, entityX509, &entityKey.PublicKey, caKey)
	require.NoError(t, err)

	certFile, err := ioutil.TempFile("", "")
	require.NoError(t, err)
	defer certFile.Close()

	// create chained PEM file with CA and entity cert
	for _, cert := range [][]byte{entityCert, caCert} {
		require.NoError(t,
			pem.Encode(certFile, &pem.Block{
				Type:  "CERTIFICATE",
				Bytes: cert,
			}),
		)
	}

	keyFile, err := ioutil.TempFile("", "")
	require.NoError(t, err)
	defer keyFile.Close()

	entityKeyBytes, err := x509.MarshalECPrivateKey(entityKey)
	require.NoError(t, err)

	require.NoError(t,
		pem.Encode(keyFile, &pem.Block{
			Type:  "ECDSA PRIVATE KEY",
			Bytes: entityKeyBytes,
		}),
	)

	cleanup := func() {
		require.NoError(t, os.Remove(certFile.Name()))
		require.NoError(t, os.Remove(keyFile.Name()))
	}

	return certFile.Name(), keyFile.Name(), cleanup
}

func DefaultLocator() storage.Locator {
	return config.NewLocator(config.Cfg{Storages: []config.Storage{{Name: "default", Path: GitlabTestStoragePath()}}})
}
