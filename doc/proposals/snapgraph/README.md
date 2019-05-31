# Proposal: SnapGraph, a storage engine for Git repositories

## High level summary

Gitaly as it exists today is a service that stores all its state on a
local filesystem. This filesystem must therefore be durable. SnapGraph
is a proposed storage engine based on SQL and object storage (e.g. S3),
which uses one or more Gitaly servers with scratch filesystems attached.

Key properties:

-   SQL decides the state of all repositories in a SnapGraph cluster
-   Git data (objects and refs) is stored as cold "snapshots" in object
    storage
-   Git reads must go through a fully extracted copy of a repository on
    a Gitaly server in the cluster
-   Git writes must go through a fully extracted copy of the repository
-   Git writes are acknowledged by Git on the Gitaly server in the
    cluster before they have been persisted to object storage; the
    system is "eventually persistent"

## Overview

## Primitives

### Git repository snapshots

In [MR 1244](https://gitlab.com/gitlab-org/gitaly/merge_requests/1244) we have an
example of how we can use Git plumbing commands to efficiently create
incremental snapshots of an entire Git repository, where each snapshot
may be stored as a single blob. We do this by combining a full dump of
the ref database of the repository, concatenated with either a full or
incremental Git packfile.

A snapshot can either be full (no "parents") or it can be incremental
relative to one or more previous snapshots (its parents). The snapshots
are incremental in exactly the same way that `git fetch` is incremental.

### Snapshot graph

Once we can make full and incremental snapshots of a repository, we can
represent that repository as a graph of snapshots where the first element must be a
full snapshot, and each later element is incremental. The simplest graph
would be a chain but in our case we also want to be able to deduplicate
Git repositories (for the "project fork" feature in GitLab), meaning we
should have a graph where incremental snapshots may depend on multiple
parents instead of a simple chain where a snapshot has at most one
parent.

Within this snapshot graph, we can think of a project as a reference to
its latest snapshot. (This is similar to how a Git branch is a reference
to its latest commit.)

### Rebuilding a repository from its snapshots

To rebuild a repository from its snapshots we must "install" all
packfiles in its graph on the Gitaly scratch server we are using. This
means more than just downloading, because a snapshot only contains the
data that goes in `.pack` files, and this data is useless without a
corresponding `.idx`. This works just the same as `git clone` and
`git fetch`, where it is up to the client (the user) to have their local
computer compute `.idx` files. Once all the packfiles in the graph of
the repository have been instantiated along with their `.idx`
companions, we bulk-import the ref database from the most recent
snapshot.

After this it is possible that we have a lot of packfiles, which is not
good for performance, so a final `git repack -ad` may be needed for
performance reasons.

### Compacting the snapshot graph

The only reason we represent a repository as a graph of multiple
snapshots is that this makes it faster to propagate writes from the
scratch Gitaly server into persistent storage (object storage). For
faster restores, and to keep the total graph size in check, we can
collapse multiple snapshots into one. This comes down to restoring the
repository in a temporary directory, up to a known snapshot. Then we
make a new full (i.e. non-incremental) snapshot from that point-in-time
copy, and replace all snapshots up to and including that point with a
single (full) snapshot.

### Snapshot graph representation

We could represent the snapshots graph as a SQL table `snapshots` with a
1-to-many relation mapping back into itself (the "parent" relation).
This table would be global to the SnapGraph cluster, and acts as
limiting factor for how many repositories and how much write activity
the cluster can sustain.

Each record in the `snapshots` table would have a corresponding object
storage blob at some immutable URL.

### Managing persistence

Similar to ongoing development for Praefect, we would have a proxy at
the "front end" of the cluster which decides if the repository in an RPC
call exists, and if so on which Gitaly scratch server the RPC will be
handled. This front end could use an event log data structure backed by
SQL where each push ("mutator" in Praefect) creates a "begin" and "end"
event in the log. Each snapshot creation job would also create a "begin" and "end"
event. The interleaving of these events in the log will then tell us the
replication state of all "dirty" repositories. Old events can be deleted, meaning that the total
number of rows in the events table is a measure for how much unpersisted
data there is in the cluster.
