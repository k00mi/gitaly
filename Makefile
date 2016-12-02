all: test build

build:
	go build -o git-daemon-server cmd/server/main.go

test:
	cd server && go test
