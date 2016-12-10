package main

import (
	"os"
	"os/signal"
	"syscall"

	"gitlab.com/gitlab-org/git-access-daemon/server"
)

func main() {
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)

	service := server.NewService()
	go func() {
		service.Serve("0.0.0.0:6666", server.CommandExecutorCallback)
	}()

	select {
	case <-ch:
		service.Stop()
	}
}
