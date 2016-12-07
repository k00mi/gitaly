package main

import (
	"os"

	"gitlab.com/gitlab-org/git-access-daemon/client"
)

func main() {
	client := client.NewClient("127.0.0.1:6666")
	defer client.Close()

	os.Args[0] = "git"
	res := client.Request(os.Args)

	if res.ExitStatus == 0 {
		os.Stdout.Write([]byte(res.Message))
	} else {
		os.Stderr.Write([]byte(res.Message))
	}

	os.Exit(res.ExitStatus)
}
