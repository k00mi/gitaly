package main

import (
	"fmt"
	"os"
	"os/exec"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: go run go-get-if-missing.go EXECUTABLE GO_PACKAGE")
		os.Exit(1)
	}
	if err := goGetIfMissing(os.Args[1], os.Args[2]); err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}
}

func goGetIfMissing(executable string, goPackage string) error {
	if _, err := exec.LookPath(executable); err == nil {
		return nil
	}

	cmd := exec.Command("go", "get", "-u", goPackage)
	cmd.Stderr = os.Stderr
	fmt.Println(cmd.Args)
	return cmd.Run()
}
