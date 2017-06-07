# Gitaly release process

- Pick a release number (x.y.z)
- Check out the master branch on your local machine
- run `_support/release x.y.z`

Where x.y.z is a semver-compliant version number.

## Experimental builds

Push the release tag to dev.gitlab.org/gitlab/gitaly. After the
passing the test suite, the tag will automatically be built and
published in https://packages.gitlab.com/gitlab/unstable.
