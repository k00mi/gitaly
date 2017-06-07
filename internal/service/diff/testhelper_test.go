package diff

import (
	"os"
	"testing"

	log "github.com/Sirupsen/logrus"

	"gitlab.com/gitlab-org/gitaly/internal/testhelper"
)

const scratchDir = "testdata/scratch"

var testRepoPath = ""

func TestMain(m *testing.M) {
	testRepoPath = testhelper.GitlabTestRepoPath()

	if err := os.MkdirAll(scratchDir, 0755); err != nil {
		log.WithError(err).Fatal("mkdirall failed")
	}

	os.Exit(func() int {
		return m.Run()
	}())
}
