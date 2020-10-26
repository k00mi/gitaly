package git

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
)

type version struct {
	major, minor, patch uint32
	rc                  bool
}

// Version returns the used git version.
func Version() (string, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var buf bytes.Buffer
	cmd, err := unsafeBareCmd(ctx, CmdStream{Out: &buf}, nil, "version")
	if err != nil {
		return "", err
	}

	if err = cmd.Wait(); err != nil {
		return "", err
	}

	out := strings.Trim(buf.String(), " \n")
	ver := strings.SplitN(out, " ", 3)
	if len(ver) != 3 {
		return "", fmt.Errorf("invalid version format: %q", buf.String())
	}

	return ver[2], nil
}

// VersionLessThan returns true if the parsed version value of v1Str is less
// than the parsed version value of v2Str. An error can be returned if the
// strings cannot be parsed.
// Note: this is an extremely simplified semver comparison algorithm
func VersionLessThan(v1Str, v2Str string) (bool, error) {
	var (
		v1, v2 version
		err    error
	)

	for _, v := range []struct {
		string
		*version
	}{
		{v1Str, &v1},
		{v2Str, &v2},
	} {
		*v.version, err = parseVersion(v.string)
		if err != nil {
			return false, err
		}
	}

	return versionLessThan(v1, v2), nil
}

func versionLessThan(v1, v2 version) bool {
	switch {
	case v1.major < v2.major:
		return true
	case v1.major > v2.major:
		return false

	case v1.minor < v2.minor:
		return true
	case v1.minor > v2.minor:
		return false

	case v1.patch < v2.patch:
		return true
	case v1.patch > v2.patch:
		return false

	case v1.rc && !v2.rc:
		return true
	case !v1.rc && v2.rc:
		return false

	default:
		// this should only be reachable when versions are equal
		return false
	}
}

func parseVersion(versionStr string) (version, error) {
	versionSplit := strings.SplitN(versionStr, ".", 4)
	if len(versionSplit) < 3 {
		return version{}, fmt.Errorf("expected major.minor.patch in %q", versionStr)
	}

	var ver version

	for i, v := range []*uint32{&ver.major, &ver.minor, &ver.patch} {
		rcSplit := strings.SplitN(versionSplit[i], "-", 2)
		n64, err := strconv.ParseUint(rcSplit[0], 10, 32)
		if err != nil {
			return version{}, err
		}

		if len(rcSplit) == 2 && strings.HasPrefix(rcSplit[1], "rc") {
			ver.rc = true
		}

		*v = uint32(n64)
	}

	if len(versionSplit) == 4 {
		if strings.HasPrefix(versionSplit[3], "rc") {
			ver.rc = true
		}
	}

	return ver, nil
}

// SupportsDeltaIslands checks if a version string (e.g. "2.20.0")
// corresponds to a Git version that supports delta islands.
func SupportsDeltaIslands(versionStr string) (bool, error) {
	v, err := parseVersion(versionStr)
	if err != nil {
		return false, err
	}

	return !versionLessThan(v, version{2, 20, 0, false}), nil
}

// SupportsReferenceTransactionHook checks if a version string corresponds to a
// Git version that supports the reference-transaction hook.
func SupportsReferenceTransactionHook(versionStr string) (bool, error) {
	v, err := parseVersion(versionStr)
	if err != nil {
		return false, err
	}

	return !versionLessThan(v, version{2, 28, 0, true}), nil
}

// NoMissingWantErrMessage checks if the git version is before Git 2.22,
// in which versions the missing objects in the wants didn't yield an explicit
// error message, but no output at all.
func NoMissingWantErrMessage() bool {
	ver, err := Version()
	if err != nil {
		return false
	}

	lt, err := VersionLessThan(ver, "2.22.0")
	return err == nil && lt
}

