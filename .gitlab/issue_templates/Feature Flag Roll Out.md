<!-- Title suggestion: [Feature flag] Enable description of feature -->

## What

Enable the `:feature_name` feature flag ...

## Owners

- Team: Gitaly
- Most appropriate slack channel to reach out to: `#g_create_gitaly`
- Best individual to reach out to: NAME

## Expectations

### What release does this feature occur in first?

### What are we expecting to happen?

### What might happen if this goes wrong?

### What can we monitor to detect problems with this?

<!--

Which dashboards from https://dashboards.gitlab.net are most relevant?
Usually you'd just like a link to the method you're changing in the
dashboard at:

https://dashboards.gitlab.net/d/000000199/gitaly-feature-status

I.e.

1. Open that URL
2. Change "method" to your feature, e.g. UserDeleteTag
3. Copy/paste the URL & change gprd to gstd to monitor staging as well as prod

-->

## Beta groups/projects

If applicable, any groups/projects that are happy to have this feature turned on early. Some organizations may wish to test big changes they are interested in with a small subset of users ahead of time for example.

- `gitlab-org/gitlab` / `gitlab-org/gitaly` projects
- `gitlab-org`/`gitlab-com` groups
- ...

## Roll Out Steps

- [ ] [Read the documentation of feature flags](https://gitlab.com/gitlab-org/gitaly/-/blob/master/doc/PROCESS.md#feature-flags)
- [ ] Add ~"featureflag::staging" to this issue
- [ ] Is the required code deployed? ([howto](https://gitlab.com/gitlab-org/gitaly/-/blob/master/doc/PROCESS.md#is-the-required-code-deployed))
- [ ] Enable on staging ([howto](https://gitlab.com/gitlab-org/gitaly/-/blob/master/doc/PROCESS.md#enable-on-staging))
- [ ] Test on staging ([howto](https://gitlab.com/gitlab-org/gitaly/-/blob/master/doc/PROCESS.md#test-on-staging))
- [ ] Ensure that documentation has been updated
- [ ] Announce on this issue an estimated time this will be enabled on GitLab.com
- [ ] Add ~"featureflag::production" to this issue
- [ ] Enable on GitLab.com by running chatops command in `#production` ([howto](https://gitlab.com/gitlab-org/gitaly/-/blob/master/doc/PROCESS.md#enable-in-production))
- [ ] Cross post chatops slack command to `#support_gitlab-com` and in your team channel
- [ ] Announce on the issue that the flag has been enabled
- [ ] Remove feature flag and add changelog entry
- [ ] Close this issue

/label ~"devops::create" ~"group::gitaly" ~"feature flag" ~"feature::maintainance" ~"Category:Gitaly" ~"section::dev" ~"featureflag::disabled"
