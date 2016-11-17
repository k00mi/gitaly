# What if we remove the shared filesystem

Once we have it all under control and it all just works (TM) we could think if this approach makes any sense.

## What would be the starting point for this?

We should have already an API for talking to git behind the Git Access Daemons in a way that we don't reach any file from git perspective anymore. Some form of RPC.
We should also have an API for the rest of the resources like LFS files, etc.

## How does it look like?

To remove shared filesystems, NFS or CephFS or whatever else, we need to take the following steps:

* Remove all mountpoints.
* Change shards in the application to also take a host and a port for git protocol.
  * We may need to add an endpoint for http API per shard depending on how we do API communication.
* With the Git Access Daemons working as the front end, we should build an HA filesystem underneath
  * Add 2 or 3 hosts with a Git Access Daemon in front.
  * Add a system like Corosync and Pacemaker in front to only allow one of the hosts to be accessible at any time.
  * Add BRBD underneath to keep the filesystems in sync.
  * Configure corosync to keep the cluster in good shape and failover to the secondary in case the primary goes away.

With this configuration the workers would have a network endpoint they talk to, which would make them independent and will allow us to use floating ips to point to hosts that will be taking the calls when they are enabled as master, otherwise they would just reject the calls completely.

![How shards could just keep growing](design/what-if/we-remove-shared-filesystems.png)

## What would we win with this?

This would remove the craziness of a system like CephFS, or the slowness of a system like NFS for something that is boring (BRBD).

In exchange this would introduce the problem of having one shard becoming a hot one. We could work a bit further and investigate what the possibility of using the primary for writes and the secondary as reads, but this would increase the complexity of the routing layer because we would need to understand when a command is a read command to redirect it to the secondary notes.

So, more complexity for more performance.
