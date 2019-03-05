# Configuring Praefect

This document describes how to configure the praefect server.

Praefect is configured via a [TOML](https://github.com/toml-lang/toml)
configuration file. The TOML file contents and location depends on how you 
installed GitLab. See: https://docs.gitlab.com/ce/administration/gitaly/

The configuration file is passed as an argument to the `praefect`
executable. This is usually done by either omnibus-gitlab or your init
script.

```
gitaly -config /path/to/config.toml
```

## Format

```toml
listen_addr = "127.0.0.1:2305"
socket_path = "/path/to/praefect.socket"

[logging]
format = "json"
level = "info"

[[gitaly_server]]
name = "default"
listen_addr = "tcp://localhost:9999"
```

An example [config toml](config.praefect.toml) is stored in this repository.
