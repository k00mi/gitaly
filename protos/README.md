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
