testdata/%.pb.go: testdata/%.proto
	go mod download gitlab.com/gitlab-org/gitaly-proto
	protoc --go_out=paths=source_relative,plugins=grpc:./testdata -I${shell go list -m -f '{{ .Dir }}' gitlab.com/gitlab-org/gitaly-proto} -I$(shell pwd)/testdata $(shell pwd)/testdata/*.proto
