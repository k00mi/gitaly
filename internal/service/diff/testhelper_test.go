package diff

import (
	"log"
	"os"
	"testing"

	"gitlab.com/gitlab-org/gitaly/internal/testhelper"
)

const scratchDir = "testdata/scratch"

var testRepoPath = ""

func TestMain(m *testing.M) {
	testRepoPath = testhelper.GitlabTestRepoPath()

	if err := os.MkdirAll(scratchDir, 0755); err != nil {
		log.Fatal(err)
	}

	os.Exit(func() int {
		return m.Run()
	}())
}
