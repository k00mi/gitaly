# Configuring Gitaly

This document describes how to configure the Gitaly server
application.

Gitaly is configured via environment variables that start with
`GITALY_`. It depends on how you installed GitLab how you should go
about setting these variables. **LINK TO DOCS SITE**

## GITALY_SOCKET_PATH

Required.

The path at which Gitaly should open a Unix socket. Example value:

```
GITALY_SOCKET_PATH=/home/git/gitlab/tmp/sockets/private/gitaly.socket
```

## GITALY_PROMETHEUS_LISTEN_ADDR

Optional.

TCP listen address for Prometheus metrics. When missing or empty, no
Prometheus listener is started.

```
GITALY_PROMETHEUS_LISTEN_ADDR=localhost:9236
```
