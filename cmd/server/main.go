package main

import (
	"gitlab.com/gitlab-org/git-access-daemon/server"
)

func main() {
	server.NewService().Serve("0.0.0.0:6666")
}
