package protoregistry

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io/ioutil"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/sirupsen/logrus"
	"gitlab.com/gitlab-org/gitaly-proto/go/gitalypb"
)

// GitalyProtoFileDescriptors is a slice of all gitaly registered file descriptors
var GitalyProtoFileDescriptors []*descriptor.FileDescriptorProto

func init() {
	for _, protoName := range gitalypb.GitalyProtos {
		gz := proto.FileDescriptor(protoName)
		fd, err := extractFile(gz)
		if err != nil {
			panic(err)
		}

		GitalyProtoFileDescriptors = append(GitalyProtoFileDescriptors, fd)
	}
}

// OpType represents the operation type for a RPC method
type OpType int

const (
	// OpUnknown = unknown operation type
	OpUnknown = iota
	// OpAccessor = accessor operation type (ready only)
	OpAccessor
	// OpMutator = mutator operation type (modifies a repository)
	OpMutator
)

// MethodInfo contains metadata about the RPC method
type MethodInfo struct {
	Operation OpType
}

// Registry contains info about RPC methods
type Registry struct {
	sync.RWMutex
	protos map[string]map[string]MethodInfo
}

// New creates a new ProtoRegistry
func New() *Registry {
	return &Registry{
		protos: make(map[string]map[string]MethodInfo),
	}
}

// RegisterFiles takes one or more descriptor.FileDescriptorProto and populates the registry with its info
func (pr *Registry) RegisterFiles(protos ...*descriptor.FileDescriptorProto) error {
	pr.Lock()
	defer pr.Unlock()
	for _, p := range protos {
		for _, serviceDescriptorProto := range p.GetService() {
			for _, methodDescriptorProto := range serviceDescriptorProto.GetMethod() {
				var mi MethodInfo

				options := methodDescriptorProto.GetOptions()

				methodDescriptorProto.GetInputType()

				if !proto.HasExtension(options, gitalypb.E_OpType) {
					logrus.WithField("service", serviceDescriptorProto.GetName()).
						WithField("method", serviceDescriptorProto.GetName()).
						Warn("grpc method missing op_type")
					continue
				}

				ext, err := proto.GetExtension(options, gitalypb.E_OpType)
				if err != nil {
					return err
				}

				opMsg, ok := ext.(*gitalypb.OperationMsg)
				if !ok {
					return fmt.Errorf("unable to obtain OperationMsg from %#v", ext)
				}

				switch opCode := opMsg.GetOp(); opCode {
				case gitalypb.OperationMsg_ACCESSOR:
					mi.Operation = OpAccessor
				case gitalypb.OperationMsg_MUTATOR:
					mi.Operation = OpMutator
				default:
					mi.Operation = OpUnknown
				}

				if _, ok := pr.protos[serviceDescriptorProto.GetName()]; !ok {
					pr.protos[serviceDescriptorProto.GetName()] = make(map[string]MethodInfo)
				}
				pr.protos[serviceDescriptorProto.GetName()][methodDescriptorProto.GetName()] = mi
			}
		}
	}

	return nil
}

// LookupMethod looks up an MethodInfo by service and method name
func (pr *Registry) LookupMethod(service, method string) (MethodInfo, error) {
	pr.RLock()
	defer pr.RUnlock()

	if _, ok := pr.protos[service]; !ok {
		return MethodInfo{}, fmt.Errorf("service not found: %v", service)
	}
	methodInfo, ok := pr.protos[service][method]
	if !ok {
		return MethodInfo{}, fmt.Errorf("method not found: %v", method)
	}
	return methodInfo, nil
}

// extractFile extracts a FileDescriptorProto from a gzip'd buffer.
// https://github.com/golang/protobuf/blob/9eb2c01ac278a5d89ce4b2be68fe4500955d8179/descriptor/descriptor.go#L50
func extractFile(gz []byte) (*descriptor.FileDescriptorProto, error) {
	r, err := gzip.NewReader(bytes.NewReader(gz))
	if err != nil {
		return nil, fmt.Errorf("failed to open gzip reader: %v", err)
	}
	defer r.Close()

	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to uncompress descriptor: %v", err)
	}

	fd := new(descriptor.FileDescriptorProto)
	if err := proto.Unmarshal(b, fd); err != nil {
		return nil, fmt.Errorf("malformed FileDescriptorProto: %v", err)
	}

	return fd, nil
}
