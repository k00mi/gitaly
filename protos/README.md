# Protobuf specifications and client libraries for Gitaly

The .proto files define the remote procedure calls for interacting
with Gitaly. We keep auto-generated client libraries for Ruby and Go
in their respective subdirectories.

Use the `_support/generate-from-proto` script from the root of the
repository to regenerate the client libraries after updating .proto
files.

See
[developers.google.com](https://developers.google.com/protocol-buffers/docs/proto3)
for documentation of the 'proto3' Protocul buffer specification
language.

## gRPC/Protobuf concepts

The core Protobuf concepts we use are rpc, service and message. We use
these to define the Gitaly **protocol**.

-   **rpc** a function that can be called from the client and that gets
    executed on the server. Belongs to a service. Can have one of four
    request/response signatures: message/message (example: get metadata for
    commit xxx), message/stream (example: get contents of blob xxx),
    stream/message (example: create new blob with contents xxx),
    stream/stream (example: git SSH session).
-   **service** a logical group of RPC's. 
-   **message** like a JSON object except it has pre-defined types.
-   **stream** an unbounded sequence of messages. In the Ruby clients
    this looks like an Enumerator.

gRPC provides an implementation framework based on these Protobuf concepts.

-   A gRPC **server** implements one or more services behind a network
    listener. Example: the Gitaly server application.
-   The gRPC toolchain automatically generates **client libraries** that
    handle serialization and connection management. Example: the Go
    client package and Ruby gem in this repository.
-   gRPC **clients** use the client libraries to make remote procedure
    calls. These clients must decide what network address to reach their
    gRPC servers on and handle connection reuse: it is possible to
    spread different gRPC services over multiple connections to the same
    gRPC server.
-   Officially a gRPC connection is called a **channel**. In the Go gRPC
    library these channels are called **client connections** because
    'channel' is already a concept in Go itself. In Ruby a gRPC channel
    is an intance of GRPC::Core::Channel. We use the word 'connection'
    in this document. The underlying transport of gRPC, HTTP/2, allows
    multiple remote procedure calls to happen at the same time on a
    single connection to a gRPC server. In principle, a multi-threaded
    gRPC client needs only one connection to a gRPC server.

## Design decisions

1.  In Gitaly's case there is one server application
    https://gitlab.com/gitlab-org/gitaly which implements all services
    in the protocol.
1.  Gitaly clients use one HTTP/2 connection per Gitaly server they
    interact with. All services used by the client share the same
    connection.
1.  Currently each Gitaly client interacts with exactly 1 Gitaly server,
    on the same host, via a Unix domain socket. In the future each
    Gitaly client will interact with many different Gitaly servers (one
    per GitLab storage shard) via TLS connections.
1.  Gitaly clients will 'cache' their gRPC connections to avoid
    connection setup overhead on each RPC call. Pitfall: the Ruby 'grpc'
    gem creates convenience methods that let you establish many new
    connections if you are not careful. To avoid this we will always use
    the `Stub.new(nil, credentials, channel_override: the_channel)`
    invocation pattern instead of
    `Stub.new('some://address', credentials)` in Ruby Gitaly clients.
1.  Gitaly uses
    [grpc.Errorf](https://godoc.org/google.golang.org/grpc#Errorf) to
    return meaningful
    [errors](https://godoc.org/google.golang.org/grpc/codes#Code) to its
    clients.
