package server

import (
	"bufio"
	"fmt"
	"net"
	"testing"
)

func TestServerStandingUp(t *testing.T) {
	service := NewService()
	address := "127.0.0.1:6666"
	go service.Serve(address)
	defer service.Stop()

	conn, err := net.Dial("tcp", address)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Write([]byte("hola hola!\n")); err != nil {
		t.Error(err)
	}
	reader := bufio.NewReader(conn)
	buffer, err := reader.ReadBytes('\n')
	if err != nil {
		t.Error(err)
	}
	conn.Close()
	fmt.Println("Received from server:", string(buffer))
}
