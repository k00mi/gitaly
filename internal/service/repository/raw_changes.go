package repository

import (
	"fmt"
	"io"
	"regexp"
	"strconv"

	"gitlab.com/gitlab-org/gitaly-proto/go/gitalypb"
	"gitlab.com/gitlab-org/gitaly/internal/git"
	"gitlab.com/gitlab-org/gitaly/internal/git/catfile"
	"gitlab.com/gitlab-org/gitaly/internal/git/rawdiff"
	"gitlab.com/gitlab-org/gitaly/internal/helper"
)

func (s *server) GetRawChanges(req *gitalypb.GetRawChangesRequest, stream gitalypb.RepositoryService_GetRawChangesServer) error {
	repo := req.Repository
	batch, err := catfile.New(stream.Context(), repo)
	if err != nil {
		return helper.ErrInternal(err)
	}

	if err := validateRawChangesRequest(req, batch); err != nil {
		return helper.ErrInvalidArgument(err)
	}

	if err := getRawChanges(stream, repo, batch, req.GetFromRevision(), req.GetToRevision()); err != nil {
		return helper.ErrInternal(err)
	}

	return nil
}

func validateRawChangesRequest(req *gitalypb.GetRawChangesRequest, batch *catfile.Batch) error {
	if req.GetRepository() == nil {
		return fmt.Errorf("repository argument must be present")
	}

	from := req.FromRevision
	if _, err := batch.Info(from); err != nil && from != git.NullSHA {
		return fmt.Errorf("invalid 'from' revision: %q", from)
	}

	to := req.ToRevision
	if _, err := batch.Info(to); err != nil && to != git.NullSHA {
		return fmt.Errorf("invalid 'to' revision: %q", to)
	}

	return nil
}

func getRawChanges(stream gitalypb.RepositoryService_GetRawChangesServer, repo *gitalypb.Repository, batch *catfile.Batch, from, to string) error {
	if to == git.NullSHA {
		return nil
	}

	if from == git.NullSHA {
		from = git.EmptyTreeID
	}

	ctx := stream.Context()

	diffCmd, err := git.Command(ctx, repo, "diff", "--raw", "-z", from, to)
	if err != nil {
		return err
	}
	p := rawdiff.NewParser(diffCmd)

	var chunk []*gitalypb.GetRawChangesResponse_RawChange

	for {
		d, err := p.NextDiff()
		if err != nil {
			if err == io.EOF {
				break // happy path
			}

			return err
		}

		change, err := changeFromDiff(batch, d)
		if err != nil {
			return err
		}
		chunk = append(chunk, change)

		const chunkSize = 50
		if len(chunk) >= chunkSize {
			resp := &gitalypb.GetRawChangesResponse{RawChanges: chunk}
			if err := stream.Send(resp); err != nil {
				return err
			}
			chunk = nil
		}
	}

	if len(chunk) > 0 {
		resp := &gitalypb.GetRawChangesResponse{RawChanges: chunk}
		if err := stream.Send(resp); err != nil {
			return err
		}
	}

	return diffCmd.Wait()
}

var zeroRegexp = regexp.MustCompile(`\A0+\z`)

const submoduleTreeEntryMode = "160000"

func changeFromDiff(batch *catfile.Batch, d *rawdiff.Diff) (*gitalypb.GetRawChangesResponse_RawChange, error) {
	resp := &gitalypb.GetRawChangesResponse_RawChange{}

	newMode64, err := strconv.ParseInt(d.DstMode, 8, 32)
	if err != nil {
		return nil, err
	}
	resp.NewMode = int32(newMode64)

	oldMode64, err := strconv.ParseInt(d.SrcMode, 8, 32)
	if err != nil {
		return nil, err
	}
	resp.OldMode = int32(oldMode64)

	if err := setOperationAndPaths(d, resp); err != nil {
		return nil, err
	}

	shortBlobID := d.DstSHA
	blobMode := d.DstMode
	if zeroRegexp.MatchString(shortBlobID) {
		shortBlobID = d.SrcSHA
		blobMode = d.SrcMode
	}

	if blobMode != submoduleTreeEntryMode {
		info, err := batch.Info(shortBlobID)
		if err != nil {
			return nil, fmt.Errorf("find %q: %v", shortBlobID, err)
		}

		resp.BlobId = info.Oid
		resp.Size = info.Size
	}

	return resp, nil
}

func setOperationAndPaths(d *rawdiff.Diff, resp *gitalypb.GetRawChangesResponse_RawChange) error {
	if len(d.Status) == 0 {
		return fmt.Errorf("empty diff status")
	}

	resp.NewPath = d.SrcPath
	resp.OldPath = d.SrcPath

	switch d.Status[0] {
	case 'A':
		resp.Operation = gitalypb.GetRawChangesResponse_RawChange_ADDED
		resp.OldPath = ""
	case 'C':
		resp.Operation = gitalypb.GetRawChangesResponse_RawChange_COPIED
		resp.NewPath = d.DstPath
	case 'D':
		resp.Operation = gitalypb.GetRawChangesResponse_RawChange_DELETED
		resp.NewPath = ""
	case 'M':
		resp.Operation = gitalypb.GetRawChangesResponse_RawChange_MODIFIED
	case 'R':
		resp.Operation = gitalypb.GetRawChangesResponse_RawChange_RENAMED
		resp.NewPath = d.DstPath
	case 'T':
		resp.Operation = gitalypb.GetRawChangesResponse_RawChange_TYPE_CHANGED
	default:
		resp.Operation = gitalypb.GetRawChangesResponse_RawChange_UNKNOWN
	}

	return nil
}
