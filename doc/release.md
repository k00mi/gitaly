# Gitaly release process

- Pick a release number (x.y.z)
- Create a CHANGELOG.md entry for x.y.z (use a merge request)
- Check out the master branch on your local machine
- Verify that the CHANGELOG for x.y.z is there
- run `_support/release x.y.z`

Where x.y.z is a semver-compliant version number.

## Experimental builds

Push the release tag to dev.gitlab.org/gitlab/gitaly. After the
passing the test suite, the tag will automatically be built and
published in https://packages.gitlab.com/gitlab/unstable.
