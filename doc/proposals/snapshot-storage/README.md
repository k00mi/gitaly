# Proposal: snapshot storage for Git repositories

## High level summary

Gitaly as it exists today is a service that stores all its state on a
local filesystem. This filesystem must therefore be durable. In this
document we describe an alternative storage system which can store
repository snapshots in SQL and object storage (e.g. S3).

Key properties:

-   SQL decides the state of all repositories in a SnapGraph cluster
-   Git data (objects and refs) is stored as cold "snapshots" in object
    storage
-   snapshots can have a "parent", so a repository is stored as a linked
    list of snapshots
-   to use the repository it must first be copied down to a local
    filesystem

## Primitives

### Git repository snapshots

In [MR 1244](https://gitlab.com/gitlab-org/gitaly/merge_requests/1244)
we have an example of how we can use Git plumbing commands to
efficiently create incremental snapshots of an entire Git repository,
where each snapshot may be stored as a single blob. We do this by
combining a full dump of the ref database of the repository,
concatenated with either a full or incremental Git packfile.

A snapshot can either be full (no "parent") or it can be incremental
relative to a previous snapshots (its parent). The snapshots are
incremental in exactly the same way that `git fetch` is incremental.

### Snapshot list

Once we can make full and incremental snapshots of a repository, we can
represent that repository as a linked list of snapshots where the first
element must be a full snapshot, and each later element is incremental.

Within this snapshot list, we can think of a project as a reference to
its latest snapshot: it is the head of the list.

### Rebuilding a repository from its snapshots

To rebuild a repository from its snapshots we must "install" all
packfiles in its list on the Gitaly server we are using. This means more
than just downloading, because a snapshot only contains the data that
goes in `.pack` files, and this data is useless without a corresponding
`.idx`. This works just the same as `git clone` and `git fetch`, where
it is up to the client (the user) to have their local computer compute
`.idx` files. Once all the packfiles in the graph of the repository have
been instantiated along with their `.idx` companions, we bulk-import the
ref database from the most recent snapshot.

After this it is possible that we have a lot of packfiles, which is not
good for performance. We also won't have a `.bitmap` file. So a final
`git repack -adb` will be needed for performance reasons.

### Compacting a snapshot list

The only reason we represent a repository as a list of multiple
snapshots is that this makes it faster to make new snapshots. For faster
restores, and to keep the total list size in check, we can collapse
multiple snapshots into one. This comes down to restoring the repository
in a temporary directory, up to a known snapshot. Then we make a new
full (i.e. non-incremental) snapshot from that point-in-time copy, and
replace all snapshots up to and including that point with a single
(full) snapshot.

### Snapshot graph representation

We could represent snapshots lists with a SQL table `snapshots` with a
1-to-1 relation mapping back into itself (the "parent" relation).

Each record in the `snapshots` table would have a corresponding object
storage blob at some immutable URL.

We need this SQL table as a catalogue of our object storage objects.

## Where to build this

Considering that Praefect will have a SQL database tracking all its
repositories, and that Praefect is aware of when repositories change and
a new snapshot is warranted, it would be a candidate for managing
snapshots.

However, we could also build this in gitlab-rails. That should work fine
for periodic snapshots, where we take snapshots regardless of whether we
know/think there was a change in the repository.

We probably don't want to build this in Gitaly itself.
