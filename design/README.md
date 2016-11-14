# Git Access Layer Daemon

_if you are lazy, there is a TL;DR section at the end_

## Introduction

In this document I will try to explain what are our main challenges in scaling GitLab from the Git perspective. It is well known already that our git access is slow, and no general purporse solution has been good enough to provide a solid experience. We've also seen than even when using CephFS we can create filesystem hot spots, which implies that pushing the problem to the filesystem layer is not enough, not even with bare metal hardware.

This can be contrasted, among other samples, simply with a look at Rugged::Repository.new performance data, where we can see that our P99 spikes up to 30 wall seconds, while the CPU time keeps in the realm of the 15 milliseconds. Pointing at filesystem access as the culprit.

Bear in mind that the aim of this effort is to provide first a relief to the main problems that affect GitLab.com availability, then to improve performance, and finally to keep digging down the performance path to reduce filesystem access to it's bare minimum.

## Initial/Current Status of Git Access

Out current storage and git access implementation is as follows

* Each worker as N CephFS or NFS mount points, as many as shards are defined.
* Each worker can take either HTTP or SSH git access.
  * HTTP git access will be handled by workhorse, shelling out to perform the command that was required by the user.
  * SSH git access is handled by ssh itself, with a standard git executable. We only wire authorization in front of it with gitlab-shell.
* Each worker can create Rugged objects or shell out either from unicorn or from sidekiq at it's own discretion.

![current status](design/img/01-current-storage-architecture.png)



TL;DR:

I think we need to make a fundamental architectural change to how we access git. Both from the application by the use of Rugged or shelling out, or just from the clients themselves running git commands in our infrastructure.

It has proven multiple times that it's easy to take GitLab.com by performing an "attack" by calling git commands on the same repo or blob, generating hotspots which neither CephFS or NFS can survive.

Additionally we have observed that our P99 access time to just create a Rugged object, which is loading and processing the git objects from disk, spikes up to 20 seconds, making it basically unusuable. We also saw that just walking through the branches of gitlab-ce requires 2.4 wall seconds. This is clearly unnacceptable.

My idea is not revolutionary. I just think that scaling is specializing, so we need to specialize our git access layer, creating a daemon which initially will only offer removing the git command execution from the workers, then it will focus on building a cache for repo's refs to offer them through an API so we can consume this from the application. Then including hot blob objects to evict CDN type attacks, and finally implementing git upload-pack protocol itself to allow us serving fetches from memory without touching the filesystem at all.

Issue about high `git cat-file blob` taking GitLab.com down: https://gitlab.com/gitlab-com/infrastructure/issues/506
