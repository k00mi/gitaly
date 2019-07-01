# `gitaly-ruby`

`gitaly-ruby` is a 'sidecar' process for the main Gitaly service. It
allows us to run legacy Ruby application code for which it would be
too risky or even infeasible to port it to Go. It was introduced to
speed up the Gitaly migration project.

## Architecture

Gitaly-ruby is a minimal Ruby gRPC service which should only receive
requests from its (Go) parent Gitaly process. The Gitaly parent
handles authentication, logging, metrics, configuration file parsing
etc.

The Gitaly parent is also responsible for starting and (if necessary)
restarting Gitaly-ruby.

## Authentication

Gitaly-ruby listens on a Unix socket in a temporary directory with
mode 0700. It runs as the same user as the Gitaly parent process.

## Testing

There are three sets of test that exercise gitaly-ruby:

- Top-level Go integration tests
- Rspec integration tests (`spec/gitaly`)
- Rspec unit tests (`spec/lib`)

If you are working on the Ruby code and you want to run the Rspec
tests only, without recompiling the Go parts then do the following:

- run `make rspec` at the top level at least once, to compile Go binaries and get the test repo;
- edit code under the current directory (`ruby`);
- run `bundle exec rspec` in the current directory.

## Development

Gitaly-ruby is only meant to be spawned by Gitaly itself. But, during
development you may want to try spawning it in a terminal yourself.
You can do that with the following incantation:

```shell
sh -c 'bin/gitaly-ruby $$ tmp/socket & wait'
```
