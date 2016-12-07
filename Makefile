all: test build

build:
	go build -o git-daemon-server cmd/server/main.go
	go build -o git-daemon-client cmd/client/main.go

test:
	cd server && go test
	cd client && go test
