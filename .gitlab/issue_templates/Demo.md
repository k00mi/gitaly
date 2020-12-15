<!--- Replace Date in title below -->

/title Demo YYYY-MM-DD

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

1. [ ] Prep:
   - [ ] Check the [latest version of this issue template](https://gitlab.com/gitlab-org/gitaly/-/blob/master/.gitlab/issue_templates/Demo.md)
   for any new steps and update this issue accordingly.
   - [ ] Checkout the latest changes from Gitaly's default branch
   - [ ] `cd _support/terraform`
   - [ ] `./create-demo-cluster`
   - [ ] `./configure-demo-cluster`
   - [ ] Sign in as admin user `root` during the demo
   - [ ] Create a new repository on the GitLab instance
   - [ ] Log into the GitLab web interface and upload license

## Features

### Repository Importer #3033

Repository importer's goal is to create any missing database records for repositories present on the disk of the primary Gitaly.

1. Prep:
   - [ ] Create a repository in the demo cluster. This ensures we have a repository on the disk we can import.
   - [ ] Stop the Praefect nodes. The import job runs when Praefect starts.
   - [ ] Truncate `virtual_storages` table. This removes the information whether the migration has been completed.
   - [ ] Truncate `repositories` table.  This removes any information about the repositories on the virtual storage.
   - [ ] Truncate `storage_repositories` table. This removes any information about repositories hosted on the Gitaly nodes.
1. [ ] Demo:
   - [ ] Start the Praefect nodes.
   - [ ] Tail Praefects' logs.
1. [ ] Verify:
   - [ ] Logs do not contain `importing repositories to database failed` indicating any import failures.
   - [ ] Logs contain `imported repositories to database` message. It should list the repository created earlier as imported.
   - [ ] Logs contain `repository importer finished` message. It should list the configured virtual storages as successfully imported.
   - [ ] Verify `repositories` table contains records for the imported repositories with generation `0`.
   - [ ] Verify `storage_repositories` records the primary containing the imported repositories on generation `0`. Secondaries might have records as well if the automatic reconciler scheduled jobs to replicate the
   repositories to them.
   - [ ] Verify `virtual_storages` table contains records with `repositories_imported` set for the successfully imported virtual storages.

### Distributed reads with caching https://gitlab.com/gitlab-org/gitaly/-/issues/3053

The goal of caching is to reduce load on the database and speed up defining up to date storages for distributing read operations among them.

1. [ ] Prep:
   - [ ] Create a repository in the demo cluster. This ensures we have a repository we can use for read/write.
   - [ ] Verify the distributed reads feature is disabled. On the GitLab node run:
     ```
     gitlab-rails console
     Feature.enabled?('gitaly_distributed_reads')
     ```
   - [ ] Verify the distributed reads cache is disabled by running
     `grep 'reads distribution caching is' /var/log/gitlab/praefect/current`
     the full message should be: `reads distribution caching is disabled because direct connection to Postgres is not set`.
1. [ ] Demo:
   - [ ] Navigate to the SQL instance in [GCloud](https://console.cloud.google.com/sql/instances?project=gitlab-internal-153318) and add addresses of the praefect instances into the set of allowed IP addresses as described at [Configuring public IP connectivity](https://cloud.google.com/sql/docs/postgres/configure-ip) page (or whitelist all IPs with `0.0.0.0/0`).
   - [ ] On each praefect node verify accessibility of the Postgres instance by running `/opt/gitlab/embedded/bin/psql -U praefect -d praefect_production -h <POSTGRESQL_SERVER_ADDRESS>`.
   - [ ] On the GitLab node enable reads distribution feature flag by running:
     ```
     gitlab-rails console
     Feature.enable('gitaly_distributed_reads')
     ```
   - [ ] Make some random operations on the repository: files creation/modification etc.
   - [ ] Verify the feature flag is set by observing `rate(gitaly_feature_flag_checks_total{flag="distributed_reads"}[5m])` metric.
   - [ ] Verify the reads distribution is working by checking grafana dashboard: `sum by (storage) (rate(gitaly_praefect_read_distribution[5m]))`.
   - [ ] Verify there is no values for `gitaly_praefect_uptodate_storages_cache_access_total` metric.
   - [ ] On each praefect node:
     - [ ] Run `vim /etc/gitlab/gitlab.rb`, change configuration by adding `praefect['database_host_no_proxy'] = '<PRAEFECT_DATABASE_IP>'` and `praefect['database_port_no_proxy'] = <PRAEFECT_DATABASE_PORT>`, and `gitlab-ctl reconfigure`. **NOTE:** You should set direct connection to Postgres database not a PgBouncer address.
     - [ ] Verify the cache is enabled by running `grep 'reads distribution caching is' /var/log/gitlab/praefect/current`
     the full message should be: `reads distribution caching is enabled by configuration`
   - [ ] Make some random operations on the repository: files creation/modification etc.
   - [ ] Verify the cache is used by checking grafana dashboard: `sum by (type) (rate(gitaly_praefect_uptodate_storages_cache_access_total[5m]))`.
   - [ ] Remember values of the `read_distribution` metric for future comparison.
   - [ ] Jump on one of the gitaly nodes and terminate it with command `gitlab-ctl stop gitaly`.
   - [ ] Make some random operations on the repository: files creation/modification etc.
   - [ ] Query for the `read_distribution` metric and compare with previous result (the terminated node should have the same value).
   - [ ] Remember values of `gitaly_praefect_uptodate_storages_cache_access_total` metric for future comparison.
   - [ ] From one of the instances login into the Postgres with `/opt/gitlab/embedded/bin/psql -U praefect -d praefect_production -h <POSTGRESQL_SERVER_ADDRESS>` and execute the query: `NOTIFY repositories_updates, '{send';`.
   - [ ] Verify each praefect instance has a log entry: `grep 'received payload can't be processed, cache disabled' /var/log/gitlab/praefect/current`.
   - [ ] Make some random operations on the repository: files creation/modification etc.
   - [ ] Query for the metric `gitaly_praefect_uptodate_storages_cache_access_total` and compare with the result saved previosuly. The `hit` metric value should not change.

## After Demo

1. [ ] Create any follow up issues discovered during the demo and assign label
   ~demo.
   - Link the issues as related to this issue
1. [ ] [Follow teardown instructions to remove demo
   resources](https://gitlab.com/gitlab-org/gitaly/-/blob/master/_support/terraform/README.md#destroying-a-demo-cluster)
1. [ ] Open a new demo issue and assign to the next demo conductor
1. [ ] Close this issue

/label ~demo ~"group::gitaly" ~"devops::create"
