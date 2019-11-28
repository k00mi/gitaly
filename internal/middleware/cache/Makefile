testdata/%.pb.go: testdata/%.proto
	protoc --go_out=paths=source_relative,plugins=grpc:./testdata -I${shell pwd}/../../../proto/ -I$(shell pwd)/testdata $(shell pwd)/testdata/*.proto
