# Gitaly code style

## Errors

### Use %v when wrapping errors

Use `%v` when wrapping errors with context.

    fmt.Errorf("foo context: %v", err)

### Keep errors short

It is customary in Go to pass errors up the call stack and decorate
them. To be a good neighbor to the rest of the call stack we should keep
our errors short.

    // Good
    fmt.Errorf("peek diff line: %v", err)

    // Too long
    fmt.Errorf("ParseDiffOutput: Unexpected error while peeking: %v", err)

### Use lower case in errors

Use lower case in errors; it is OK to preserve upper case in names.

### Errors should stick to the facts

It is tempting to write errors that explain the problem that occurred.
This can be appropriate in some end-user facing situations, but it is
never appropriate for internal error messages. When your
interpretation is wrong it puts the reader on the wrong track.

Stick to the facts. Often it is enough to just describe in a few words
what we were trying to do.

### Use %q when interpolating strings

Unless it would lead to incorrect results, always use `%q` when
interpolating strings. The `%q` operator quotes strings and escapes
spaces and non-printable characters. This can save a lot of debugging
time.

## Tests

### Table-driven tests

We like table-driven tests ([Cheney blog post], [Golang wiki]).

-   It should be clear from error messages which test case in the table
    is failing. Consider giving test cases a `name` attribute and
    including that name in every error message inside the loop.
-   Use `t.Errorf` inside a table test loop, not `t.Fatalf`: this
    provides more feedback when fixing failing tests.

  [Cheney blog post]: https://dave.cheney.net/2013/06/09/writing-table-driven-tests-in-go
  [Golang wiki]: https://github.com/golang/go/wiki/TableDrivenTests
