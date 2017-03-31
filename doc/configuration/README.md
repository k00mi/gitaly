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
