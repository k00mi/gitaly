// Package glsql provides integration with SQL database. It contains a set
// of functions and structures that help to interact with SQL database and
// to write tests to check it.

// A simple unit tests do not require any additional dependencies except mocks.
// The tests to check interaction with a database require database to be up and
// running.
// You must provide PGHOST and PGPORT environment variables to run the tests.
// PGHOST - is a host of the Postgres database to connect to.
// PGPORT - is a port which is used by Postgres database to listen for incoming
// connections.
//
// If you use Gitaly inside GDK or want to reuse Postgres instance from GDK
// navigate to gitaly folder from GDK root and run command:
// $ gdk env
// it should print PGHOST and PGPORT for you.

// To check if everything configured properly run the command:
//
// $ PGHOST=<host of db instance> \
//   PGPORT=<port of db instance> \
//   go test \
//    -tags=postgres \
//    -v \
//    -count=1 \
//    gitlab.com/gitlab-org/gitaly/internal/praefect/datastore/glsql \
//    -run=^TestOpenDB$
//
// Once it is finished successfully you can be sure other tests would be able to
// connect to the database and interact with it.
//
// As you may note there is a special build tag `postgres` in the `go` command
// above. This build tag distinguishes tests that depend on the database from those
// which are not. When adding a new tests with a dependency to database please add
// this build tag to them. The example how to do this could be found in
// internal/praefect/datastore/glsql/postgres_test.go file.

package glsql
