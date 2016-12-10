package client

import (
	"bufio"
	"encoding/json"
	"log"
	"net"

	"gitlab.com/gitlab-org/git-access-daemon/server"
)

type Client struct {
	conn net.Conn
}

func NewClient(serviceAddress string) *Client {
	conn, err := net.Dial("tcp", serviceAddress)
	if err != nil {
		log.Fatalln(err)
	}

	return &Client{conn}
}

func (client *Client) Request(cmd []string) server.CmdResponse {
	_, err := client.conn.Write(append(makeRequest(cmd), "\n"...))
	if err != nil {
		log.Fatalln(err)
	}

	reader := bufio.NewReader(client.conn)
	buffer, err := reader.ReadBytes('\n')
	if err != nil {
		log.Fatalln(err)
	}

	return parseResponse(buffer)
}

func (client *Client) Close() {
	client.conn.Close()
}

func makeRequest(cmd []string) []byte {
	req := server.CmdRequest{
		Cmd: cmd,
	}

	buf, err := json.Marshal(&req)
	if err != nil {
		log.Fatalln("Failed marshalling a JSON request")
	}

	return buf
}

func parseResponse(rawResponse []byte) server.CmdResponse {
	res := server.CmdResponse{}

	err := json.Unmarshal(rawResponse, &res)
	if err != nil {
		log.Fatalln("Failed parsing response")
	}

	return res
}
