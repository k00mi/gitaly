# Gitaly release process

Releases are marked by annotated Git tags. To create a new release
run:

```
_support/release x.y.z
```

Where x.y.z is a semver-compliant version number.

## Experimental builds

Push the release tag to dev.gitlab.org/gitlab/gitaly. After the
passing the test suite, the tag will automatically be built and
published in https://packages.gitlab.com/gitlab/unstable.
