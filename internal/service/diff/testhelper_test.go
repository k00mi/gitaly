package diff

import (
	"os"
	"testing"

	log "github.com/Sirupsen/logrus"

	"gitlab.com/gitlab-org/gitaly/internal/testhelper"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"
)

const scratchDir = "testdata/scratch"

var testRepo *pb.Repository

func TestMain(m *testing.M) {
	testRepo = testhelper.TestRepository()

	if err := os.MkdirAll(scratchDir, 0755); err != nil {
		log.WithError(err).Fatal("mkdirall failed")
	}

	os.Exit(func() int {
		return m.Run()
	}())
}
