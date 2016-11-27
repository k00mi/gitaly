package main

import (
	"gitlab.com/gitlab-org/git-access-daemon/server"
)

func main() {
	server.Listen("tcp", "0.0.0.0:6666")
}
