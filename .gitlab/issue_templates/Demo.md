<!--- Replace Date in title below -->

/title Demo YYYY-MM-DD

<!-- Replace due date below with the date of the demo -->

/due YYYY-MM-DD

<!--
## Contributing

When adding new feature demonstrations to the script, follow these guidelines.

For each feature you are verifying, add an H3 section with a link to the issue
to the `## Features` section.

Always add new features near the bottom of this section. This way older issues
will float to the top and allow them to be prioritized during the demo.

Make sure you break down steps into the following sections:

1. prep steps - these are steps needed to correctly set up your demonstration.
   These steps are okay for the demo runner to perform before the start of the
   demo call.
1. demo steps - these are the steps to perform during the demo call to show
   how the feature works 
1. verify steps - these are the expected observations required to be seen
   in order to verify the prep or feature works as expected

Ideally, all setup steps should before the exercise steps (when possible).
Demo and verification steps may interleave as needed. For example, the
following structure is okay:

1. Prep
1. Prep
1. Verify
1. Prep
1. Demo 
1. Verify
1. Demo
1. Demo
1. Verify
1. Verify

Along with the H3 section, it might look like this:

```
### #1234

1. [ ] Prep: install thingy
1. [ ] Verify: thingy works
1. [ ] Prep: turn on gizmo
1. [ ] Demo: press red button
1. [ ] Verify: world should explode
```

When your feature passes all verification steps, submit an MR to remove
it from this issue template.

-->

This issue is used to conduct a demo for exhibiting and verifying new behavior
for Gitaly and Praefect. Before the demo, run all `Prep:` steps. During the
demo, run through all remaining `Demo:` and `Verify` steps. Check each
step as completed or verified. Do not check a `Verify:` step if it does not
succeed. 

## General Setup

1. [ ] Prep: Check the
   [latest version of this issue template](https://gitlab.com/gitlab-org/gitaly/-/blob/master/.gitlab/issue_templates/Demo.md)
   for any new steps and update this issue accordingly.
1. [ ] Prep: [Follow instructions to setup and configure a GitLab instance with
   a Praefect
   cluster](https://gitlab.com/gitlab-org/gitaly/-/blob/master/_support/terraform/README.md).
1. [ ] Prep: Log into the GitLab web interface and upload license

## Features
<!-- Keep older features near the top of this section, add new features at
the bottom -->

## After Demo

1. [ ] Open a new demo issue and assign to the next demo runner.
1. [ ] Create any follow up issues discovered during the demo and assign label
   ~demo.
1. [ ] [Follow teardown instructions to remove demo
   resources](https://gitlab.com/gitlab-org/gitaly/-/blob/master/_support/terraform/README.md#destroying-a-demo-cluster)
1. [ ] Close this issue

/label ~demo ~"group::gitaly" ~"devops::create"