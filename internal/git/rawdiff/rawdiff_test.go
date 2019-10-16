package rawdiff

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParser(t *testing.T) {
	testCases := []struct {
		desc string
		in   string
		out  *Diff
	}{
		{
			desc: "one path",
			in:   ":000000 100644 0000000 c74175a A\000CHANGELOG\000",
			out: &Diff{
				SrcMode: "000000",
				DstMode: "100644",
				SrcSHA:  "0000000",
				DstSHA:  "c74175a",
				Status:  "A",
				SrcPath: "CHANGELOG",
			},
		},
		{
			desc: "two paths (C)",
			in:   ":000000 100644 0000000 c74175a C\000CHANGELOG\000foobar\000",
			out: &Diff{
				SrcMode: "000000",
				DstMode: "100644",
				SrcSHA:  "0000000",
				DstSHA:  "c74175a",
				Status:  "C",
				SrcPath: "CHANGELOG",
				DstPath: "foobar",
			},
		},
		{
			desc: "two paths (R)",
			in:   ":000000 100644 0000000 c74175a R\000CHANGELOG\000foobar\000",
			out: &Diff{
				SrcMode: "000000",
				DstMode: "100644",
				SrcSHA:  "0000000",
				DstSHA:  "c74175a",
				Status:  "R",
				SrcPath: "CHANGELOG",
				DstPath: "foobar",
			},
		},
		{
			desc: "special characters",
			in:   ":000000 100644 0000000 c74175a A\000encoding/テスト.txt\000",
			out: &Diff{
				SrcMode: "000000",
				DstMode: "100644",
				SrcSHA:  "0000000",
				DstSHA:  "c74175a",
				Status:  "A",
				SrcPath: "encoding/テスト.txt",
			},
		},
		{
			desc: "status with score",
			in:   ":000000 100644 0000000 c74175a T100\000CHANGELOG\000",
			out: &Diff{
				SrcMode: "000000",
				DstMode: "100644",
				SrcSHA:  "0000000",
				DstSHA:  "c74175a",
				Status:  "T100",
				SrcPath: "CHANGELOG",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			r := strings.NewReader(tc.in)
			p := NewParser(r)

			d, err := p.NextDiff()
			require.NoError(t, err)

			require.Equal(t, tc.out, d)

			_, err = p.NextDiff()
			require.Equal(t, io.EOF, err)
		})
	}
}
