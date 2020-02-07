// +build postgres

package glsql

import (
	"log"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	code := m.Run()
	// Clean closes connection to database once all tests are done
	if err := Clean(); err != nil {
		log.Fatalln(err, "database disconnection failure")
	}
	os.Exit(code)
}
