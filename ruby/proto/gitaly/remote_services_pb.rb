# Generated by the protocol buffer compiler.  DO NOT EDIT!
# Source: remote.proto for package 'gitaly'

require 'grpc'
require 'remote_pb'

module Gitaly
  module RemoteService
    class Service

      include GRPC::GenericService

      self.marshal_class_method = :encode
      self.unmarshal_class_method = :decode
      self.service_name = 'gitaly.RemoteService'

      rpc :AddRemote, Gitaly::AddRemoteRequest, Gitaly::AddRemoteResponse
      rpc :FetchInternalRemote, Gitaly::FetchInternalRemoteRequest, Gitaly::FetchInternalRemoteResponse
      rpc :RemoveRemote, Gitaly::RemoveRemoteRequest, Gitaly::RemoveRemoteResponse
      rpc :UpdateRemoteMirror, stream(Gitaly::UpdateRemoteMirrorRequest), Gitaly::UpdateRemoteMirrorResponse
      rpc :FindRemoteRepository, Gitaly::FindRemoteRepositoryRequest, Gitaly::FindRemoteRepositoryResponse
      rpc :FindRemoteRootRef, Gitaly::FindRemoteRootRefRequest, Gitaly::FindRemoteRootRefResponse
      rpc :ListRemotes, Gitaly::ListRemotesRequest, stream(Gitaly::ListRemotesResponse)
    end

    Stub = Service.rpc_stub_class
  end
end
