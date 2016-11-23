all: build

build: 
	go build -o git-daemon-server cmd/server/main.go
