# Configuring Gitaly

This document describes how to configure the Gitaly server
application.

Gitaly is configured via environment variables that start with
`GITALY_`. It depends on how you installed GitLab how you should go
about setting these variables. See:
https://docs.gitlab.com/ce/administration/gitaly/

## GITALY_SOCKET_PATH

Required unless GITALY_LISTEN_ADDR is set.

A path at which Gitaly should open a Unix socket. Example value:

```
GITALY_SOCKET_PATH=/home/git/gitlab/tmp/sockets/private/gitaly.socket
```

## GITALY_LISTEN_ADDR

Required unless GITALY_SOCKET_PATH is set.

TCP address for Gitaly to listen on. Note: at the moment Gitaly does
not offer any form of authentication. When you use a TCP listener you
must use firewalls or other network-based security to restrict access
to Gitaly.

Example value:

```
GITALY_LISTEN_ADDR=localhost:1234
```

## GITALY_PROMETHEUS_LISTEN_ADDR

Optional.

TCP listen address for Prometheus metrics. When missing or empty, no
Prometheus listener is started.

```
GITALY_PROMETHEUS_LISTEN_ADDR=localhost:9236
```

## Configuration file

Gitaly also takes a path to a config-file as a command-line argument.

```
./gitaly /path/to/config.toml
```

*NOTE*: Environment variables currently takes percedence over the configuration
file. But environment variables will be depricated at some point.

### Format

```toml
socket_path = "/path/to/gitaly.sock"
listen_addr = ":8081"
prometheus_listen_addr = ":9236"

[[storage]]
path = "/path/to/storage/repositories"
name = "my_shard"

# Gitaly may serve from multiple storages
#[[storage]]
#name = "other_storage"
#path = "/path/to/other/repositories"
```

#### Explanation

|name|type|required|notes|
|----|----|--------|-----|
|socket_path|string|see notes|A path which gitaly should open a Unix socket. Required unless listen_addr is set|
|listen_addr|string|see notes|TCP address for Gitaly to listen on (See #GITALY_LISTEN_ADDR). Required unless socket_path is set|
|prometheus_listen_addr|string|no|TCP listen address for Prometheus metrics. If not set, no Prometheus listener is started|
|storage|array|yes|An array of storage shards|

##### Storage

|name|type|required|notes|
|----|----|--------|-----|
|path|string|yes|Path to storage shard|
|name|string|yes|Name of storage shard|
