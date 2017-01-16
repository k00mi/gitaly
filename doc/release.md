# Gitaly release process

Releases are marked by annotated Git tags. They must be of the form
`vX.Y.Z`.

## Experimental builds

Push the release tag to dev.gitlab.org/gitlab/gitaly. After the
passing the test suite, the tag will automatically be built and
published in https://packages.gitlab.com/gitlab/unstable.
