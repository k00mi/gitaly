// +build postgres

package nodes

import (
	"log"
	"os"
	"testing"

	"gitlab.com/gitlab-org/gitaly/internal/praefect/datastore/glsql"
	"gitlab.com/gitlab-org/gitaly/internal/testhelper"
)

func TestMain(m *testing.M) {
	os.Exit(testMain(m))
}

func testMain(m *testing.M) int {
	defer testhelper.MustHaveNoChildProcess()
	cleanup := testhelper.Configure()
	defer cleanup()

	// Clean closes connection to database once all tests are done
	defer func() {
		if err := glsql.Clean(); err != nil {
			log.Fatalln(err, "database disconnection failure")
		}
	}()

	return m.Run()
}

func getDB(t testing.TB) glsql.DB { return glsql.GetDB(t, "nodes") }
