# Git Access Layer Daemon

_if you are lazy, there is a TL;DR section at the end_

## Introduction

In this document I will try to explain what are our main challenges in scaling GitLab from the Git perspective. It is well known already that our [git access is slow,](https://gitlab.com/gitlab-com/infrastructure/issues/351) and no general purporse solution has been good enough to provide a solid experience. We've also seen than even when using CephFS we can create filesystem hot spots, which implies that pushing the problem to the filesystem layer is not enough, not even with bare metal hardware.

This can be contrasted, among other samples, simply with a look at Rugged::Repository.new performance data, where we can see that our P99 spikes up to 30 wall seconds, while the CPU time keeps in the realm of the 15 milliseconds. Pointing at filesystem access as the culprit.

![rugged.new timings](design/img/rugged-new-timings.png)


Bear in mind that the aim of this effort is to provide first a relief to the main problems that affect GitLab.com availability, then to improve performance, and finally to keep digging down the performance path to reduce filesystem access to it's bare minimum.

## Initial/Current Status of Git Access

Out current storage and git access implementation is as follows

* Each worker as N CephFS or NFS mount points, as many as shards are defined.
* Each worker can take either HTTP or SSH git access.
  * HTTP git access will be handled by workhorse, shelling out to perform the command that was required by the user.
  * SSH git access is handled by ssh itself, with a standard git executable. We only wire authorization in front of it with gitlab-shell.
* Each worker can create Rugged objects or shell out either from unicorn or from sidekiq at it's own discretion.

![current status](design/img/01-current-storage-architecture.png)

One of the main issues we face from production is that this "everyone can do whatever it wants" approach introduces a lot of stress into the filesystem, with the worse result being taking GitLab.com down.

### GitLab.com down when NFS goes down

In this issue we concluded that since we have access to Rugged objects or shelling out, when the filesystem goes down, the whole site goes down. We just don't have a concept of degraded mode at all.

### GitLab.com goes down when it is used as a CDN

* https://gitlab.com/gitlab-com/operations/issues/199
* https://gitlab.com/gitlab-com/infrastructure/issues/506

We have seen this multiple times, the pattern is as follows:

1. Someone distributes a git committed file at GitLab.com and offers access through the API
1. This file is requested by multiple people at roughly the same time (Hackers News effect - AKA slashdot effect)
1. We see increased load in the workers
1. Concidently we see high IO wait
1. We detect hundreds of `git cat-file blob` processes running on the affected workers.
1. GitLab.com is effectively down.

#### Graphs to show at least 2 events like these

##### Event 1

![Workers under heavy load](design/img/git-cat-file-workers.png)

![Decrease on GitLab.com connections](design/img/git-cat-file-connections.png)

##### Event 2

Not necesarily related to git-cat-file-blob, but git was found with a smoking gun

![Workers under heavy load](design/img/git-high-load-workers-load.png)

![Workers with high IOWait](design/img/git-high-load-workers-io-wait.png)

![Drop in GitLab.com Connections](design/img/git-high-load-connections-down.png)

![High git processes count](design/img/git-high-load-process-count.png)

#### What does this mean from the architectural perspective?

These events have 2 possible reads

1. Git can be load and IO intensive, by keeping Git executions in the workers the processes that serve GitLab.com will be competing for these resources, in many cases just loosing the fight, or getting to a locked state if what these processes are trying to do is in fact reach the git storage.
1. Accesses like these create hot spots in the filesystem, we have seen this happening in NFS and in CephFS, more pronounced in CephFS given the latency constraints from the cloud.

![Hot spot architectural design](design/img/02-high-stress-single-point.png)

### GitLab.com git access is slow

I don't think I need to add a lot of data here, it's wildly known and we have plenty data. I'll skip it for now.

## OMG! what can we do?

Run to the hills!

## TL;DR:

I think we need to make a fundamental architectural change to how we access git. Both from the application by the use of Rugged or shelling out, or just from the clients themselves running git commands in our infrastructure.

It has proven multiple times that it's easy to take GitLab.com by performing an "attack" by calling git commands on the same repo or blob, generating hotspots which neither CephFS or NFS can survive.

Additionally we have observed that our P99 access time to just create a Rugged object, which is loading and processing the git objects from disk, spikes over 30 seconds, making it basically unusuable. We also saw that just walking through the branches of gitlab-ce requires 2.4 wall seconds. This is clearly unnacceptable.

My idea is not revolutionary. I just think that scaling is specializing, so we need to specialize our git access layer, creating a daemon which initially will only offer removing the git command execution from the workers, then it will focus on building a cache for repo's refs to offer them through an API so we can consume this from the application. Then including hot blob objects to evict CDN type attacks, and finally implementing git upload-pack protocol itself to allow us serving fetches from memory without touching the filesystem at all.

