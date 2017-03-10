# Gitaly code style

## Errors

### Use %v when wrapping errors

Use `%v` when wrapping errors with context.

    return fmt.Errorf("foo context: %v", err)

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
