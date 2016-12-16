package messaging

import (
	"log"

	proto "github.com/golang/protobuf/proto"
)

func ParseMessage(rawMsg []byte) (*Message, error) {
	msg := &Message{}

	err := proto.Unmarshal(rawMsg, msg)
	return msg, err
}

func NewCommandMessage(environ []string, name string, args ...string) []byte {
	msg := &Message{
		Type: "command",
		Payload: &Message_Command{
			&Command{name, args, environ},
		},
	}

	return marshal(msg)
}

func NewInputMessage(input []byte) []byte {
	msg := &Message{
		Type: "stdin",
		Payload: &Message_Input{
			&Input{input},
		},
	}

	return marshal(msg)
}

func NewOutputMessage(outputType string, output []byte) []byte {
	msg := &Message{
		Type: outputType,
		Payload: &Message_Output{
			&Output{output},
		},
	}

	return marshal(msg)
}

func NewExitMessage(exitStatus int32) []byte {
	msg := &Message{
		Type: "exit",
		Payload: &Message_Exit{
			&Exit{exitStatus},
		},
	}

	return marshal(msg)
}

func marshal(msg *Message) []byte {
	buffer, err := proto.Marshal(msg)
	if err != nil {
		log.Fatalln("Failed marshalling a Protobuf message")
	}

	return buffer
}
