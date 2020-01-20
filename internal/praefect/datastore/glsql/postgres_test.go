// +build postgres

package glsql

import (
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/gitaly/internal/praefect/config"
)

func TestOpenDB(t *testing.T) {
	dbCfg := config.DB{
		Host: os.Getenv("PGHOST"),
		Port: func() int {
			pgPort := os.Getenv("PGPORT")
			port, err := strconv.Atoi(pgPort)
			require.NoError(t, err, "failed to parse PGPORT %q", pgPort)
			return port
		}(),
		DBName:  "postgres",
		SSLMode: "disable",
	}

	t.Run("failed to ping because of incorrect config", func(t *testing.T) {
		badCfg := dbCfg
		badCfg.Host = "not-existing.com"
		_, err := OpenDB(badCfg)
		require.Error(t, err, "opening of DB with incorrect configuration must fail")
	})

	t.Run("connected with proper config", func(t *testing.T) {
		db, err := OpenDB(dbCfg)
		require.NoError(t, err, "opening of DB with correct configuration must not fail")
		require.NoError(t, db.Close())
	})
}
