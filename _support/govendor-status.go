package main

import (
	"fmt"
	"os"
	"os/exec"
)

const (
	govendorExcutable = "govendor"
	govendorSource    = "github.com/kardianos/govendor"
)

func main() {
	if err := doStuff(); err != nil {
		fmt.Printf("error: %v", err)
		os.Exit(1)
	}
}

func doStuff() error {
	if _, err := exec.LookPath(govendorExcutable); err != nil {
		if err := cmd("go", "get", "-u", govendorSource).Run(); err != nil {
			return err
		}
	}
	return cmd(govendorExcutable, "status").Run()
}

func cmd(name string, arg ...string) *exec.Cmd {
	result := exec.Command(name, arg...)
	result.Stderr = os.Stderr
	fmt.Println(result.Args)
	return result
}
