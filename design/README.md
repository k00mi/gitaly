# Git Access Layer Daemon

_if you are lazy, there is a TL;DR section at the end_

## Introduction

In this document I will try to explain what are our main challenges in scaling GitLab from the Git perspective. It is well known already that our [git access is slow,](https://gitlab.com/gitlab-com/infrastructure/issues/351) and no general purpose solution has been good enough to provide a solid experience. We've also seen than even when using CephFS we can create filesystem hot spots, which implies that pushing the problem to the filesystem layer is not enough, not even with bare metal hardware.

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
1. Coincidently we see high IO wait
1. We detect hundreds of `git cat-file blob` processes running on the affected workers.
1. GitLab.com is effectively down.

#### Graphs to show at least 2 events like these

##### Event 1

![Workers under heavy load](design/img/git-cat-file-workers.png)

![Decrease on GitLab.com connections](design/img/git-cat-file-connections.png)

##### Event 2

Not necessarily related to git-cat-file-blob, but git was found with a smoking gun

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

## OMG! What can we do?

I think we need to attack all these problems as a whole by isolating and abstracting Git access, first from the worker hosts, then from the application, and then specializing this access layer to provide a fast implementation of the git protocol that does not depends so much in filesystem speed by leveraging memory use for the critical bits.

> It sounds scary and I don't know what to do or how to deal with it!

Let's start with the basics, we need to separate the urgent (availability) from the important (performance)

### Stage one: bulkheads for availability

The first step will be to just remove all the git processes from the workers into a specific fleet of git workers, it's good enough to have just a couple of those as the downside of being attacked will be that these hosts will be under heavy load, not so the workers.

This design will allow the application to fail gracefully when we are being under heavy stress and will allow us to start specializing this git access layer, even including throttling, rate limiting per repo, and monitoring of git usage (something we still don't have) from the minute 0.

The way we will move the process would be by providing a simple client that will simply forward the git command that is being sent either through SSH or HTTPS to a daemon that will be listening on these workers. This daemon will simply spawn a thread (or go routine) where it will execute this git command sending stdin/out to the original client, acting as a transparent proxy for the git access.

The goal here is simply to remove the git execution from the workers, and to build the ground work to keep moving forward.

![Bulkheads architecture](design/img/03-low-stress-single-point.png)

This design will allow us to start walking in the direction of removing git access from the application, but let's keep moving too how it would look like.

### Stage two: specialization for performance

Once we have availability taken care of we will need to start working on the performance side. As I commented previously the main performance hog we are seeing comes from filesystem access as a whole, so that's what we should take care of.

When a Rugged::Repository object is created what happens is that all the refs are loaded in memory so this object can reply to things like _is this a valid repo?_ and _give me the branches names_. For performance reasons these refs could be packed all in one file or they could be spread through multiple files.

Each option has its own benefits and drawbacks. A single file is not nicely managed neither by NFS or CephFS and can create locks contention given enough concurrent access. Multiple files on the other hand translate in multiple file accesses which increases pressure in the filesystem itself, just by opening, reading an closing a lot of tiny files.

I think we need to remove this pressure point as a whole and pull it out of the filesystem completely. It happens that our read to write ration is massive: we read way, way, way more times than we write. A project like gitlab-ce that is under heavy development could have something like a couple of hundred writes, but it will tens of thousands reads per day.

So, I propose that we use this caching layer to load the refs into a memory hashmap. This makes sense because git behaves as a hashmap itself: refs are both keys (branches, tags, the HEAD) and values (the SHA of the git object). Just by caching this in memory we could do the following:

* Serve the refs list from this cache, preventing calls for 'advertise refs' from git clients to hit the filesystem at all.
* Serve branches, tags and last commits through an HTTP API that can be consumed by the workers.
* Start caching specifically requested blobs in memory for quick access (to improve the cat-file blob case even further)
* Start caching diffs to improve diff access times.
* Remove all Rugged::Repository and shell outs from the application by using this API.
* Remove git mount points from the application and mount them in this caching layer instead to completely isolate workers from git storage failures.

![High level architecture](design/img/04-git-access-layer-high-level-architecture.png)

> But that sounds like a risky business, how are we going to invalidate the cache? How are we going to control memory usage?

Glad you ask!

#### Keeping Memory Down

Let's keep it boring and use an LRU Pages. In a way this will enable having a way of garbage collecting refs and blobs that are simply not being requested by clients.

The way it works is that we keep a linked list of hashmaps. From most to least recent. Every X time (any event) we evict the last page from memory deallocating all the keys and values that are there and we add a new _page_ (which is a hashmap) at the beginning of the list

The way we promote elements from one page to the other is by client request: when a client requests a value we look for it in the first page, then the second, etc. Until we find it, when we do we promote it to the first list keeping these values in memory.

This way, keys that are not being requested will naturally go down the line and will be evicted, pages that are required will be living the initial pages all the time.

This way we will have a O(n) search time (being N the number of pages, which could be 2, so it would end up being constant time) for a given key.

When a page is not found anywhere we will pull it from disk into the first page, starting the cycle again.

![Pulling Data](design/img/05-git-access-layer-pulling-data.png)

This extreme simplicity will allow us to play with the right approach for keeping memory down, some ideas:

* We could have different strategies for eviction, heartbeat based, memory threshold, number of keys in the first page, whatever we can think of.
* We could even enforce keeping certain projects always in memory, keeping the cache warm for specific high usage projects.


> Ok, but how are we going to not make it a single point of failure?

Thanks for asking! I was just going to explain that :)

#### Ephemeral nature

This git access layer has to be ephemeral, it is pointless to depend on a cache that you can't evict. For this reason I don't think that we should ever depend on a specific instance of this layer ever.

The way I see it is that we should be able to scale up and down these daemons as we need, so they should not depend on each other, but they should talk.

Cache invalidation should happen whenever we pipe a write command (a git push), when this happens the following events should roll out

* The client starts a push
* The git access daemon gets the call and passes it through to the storage layer
* On write finish
  * The git access daemon evicts it's own cache for the pushed repo
  * The daemon then published in a Redis pub/sub topic the name of the repo
  * All the other daemons, which are subscribed to the topic from the beginning get the message and evict their own cache
* The git access daemon finishes the write and returns the answer to the git client.

![Pushing Data](design/img/06-git-access-layer-pushing-data.png)

Of course there are details to fully flesh out. Particularly failure modes for when a write fails or when the daemon crashes while performing this write.

For brevity I rather not do it here, but just to throw some food for thought: we could be really aggressive to evict caches when we get a write by pushing a delayed queue into Redis and keeping it from happening with a heartbeat until we finish the write, worse case scenario we would be evicting a cache that is actually valid and it would be reloaded on a client request.


### Stage three: let's start talking protocols

Since we got this far, I would like to take this opportunity to start talking a bit on how do I envision this in the long run.

Up to this point we probably removed a lot of the pain points of performance in the application by keeping those pesky refs in memory, but that is not all there is. If we want to keep scaling and keep getting more and more load we need to understand how our clients behave, we need to be smarter and we need to be one step ahead of them.

Just basing myself in anecdotal evidence.


![Final architecture](design/img/07-git-access-layer-final-architecture.png)


## TL;DR:

I think we need to make a fundamental architectural change to how we access git. Both from the application by the use of Rugged or shelling out, or just from the clients themselves running git commands in our infrastructure.

It has proven multiple times that it's easy to take GitLab.com by performing an "attack" by calling git commands on the same repo or blob, generating hotspots which neither CephFS or NFS can survive.

Additionally we have observed that our P99 access time to just create a Rugged object, which is loading and processing the git objects from disk, spikes over 30 seconds, making it basically unusable. We also saw that just walking through the branches of gitlab-ce requires 2.4 wall seconds. This is clearly unacceptable.

My idea is not revolutionary. I just think that scaling is specializing, so we need to specialize our git access layer, creating a daemon which initially will only offer removing the git command execution from the workers, then it will focus on building a cache for repo's refs to offer them through an API so we can consume this from the application. Then including hot blob objects to evict CDN type attacks, and finally implementing git upload-pack protocol itself to allow us serving fetches from memory without touching the filesystem at all.

Now go and take a look at the images, they explain it all.
