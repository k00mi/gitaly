package ref

import (
	pb "gitlab.com/gitlab-org/gitaly-proto/go"
)

func appendRef(refs [][]byte, p []byte) ([][]byte, int) {
	ref := make([]byte, len(p))
	size := copy(ref, p)
	return append(refs, ref), size
}

type refNamesSender interface {
	sendRefs([][]byte) error
}

type refNamesWriter struct {
	refNamesSender
	MaxMsgSize int
	refsSize   int
	refs       [][]byte
}

func (w *refNamesWriter) Flush() error {
	if len(w.refs) == 0 { // No message to send, just return
		return nil
	}

	if err := w.refNamesSender.sendRefs(w.refs); err != nil {
		return err
	}

	// Reset the message
	w.refs = nil
	w.refsSize = 0

	return nil
}

func (w *refNamesWriter) AddRef(p []byte) error {
	refs, size := appendRef(w.refs, p)
	w.refsSize += size
	w.refs = refs

	if w.refsSize > w.MaxMsgSize {
		return w.Flush()
	}

	return nil
}

type branchesSender struct {
	stream pb.Ref_FindAllBranchNamesServer
}

func (w branchesSender) sendRefs(refs [][]byte) error {
	return w.stream.Send(&pb.FindAllBranchNamesResponse{Names: refs})
}

type tagsSender struct {
	stream pb.Ref_FindAllTagNamesServer
}

func (w tagsSender) sendRefs(refs [][]byte) error {
	return w.stream.Send(&pb.FindAllTagNamesResponse{Names: refs})
}

func newFindAllBranchNamesWriter(stream pb.Ref_FindAllBranchNamesServer, maxMsgSize int) refNamesWriter {
	return refNamesWriter{
		refNamesSender: branchesSender{stream},
		MaxMsgSize:     maxMsgSize,
	}
}

func newFindAllTagNamesWriter(stream pb.Ref_FindAllTagNamesServer, maxMsgSize int) refNamesWriter {
	return refNamesWriter{
		refNamesSender: tagsSender{stream},
		MaxMsgSize:     maxMsgSize,
	}
}
