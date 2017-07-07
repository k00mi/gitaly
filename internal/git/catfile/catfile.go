package catfile

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"
)

// ObjectInfo represents a header returned by `git cat-file --batch`
type ObjectInfo struct {
	Oid  string
	Type string
	Size int64
}

// ParseObjectInfo reads and parses one header line from `git cat-file --batch`
func ParseObjectInfo(stdout *bufio.Reader) (*ObjectInfo, error) {
	infoLine, err := stdout.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("read info line: %v", err)
	}

	infoLine = strings.TrimSuffix(infoLine, "\n")
	if strings.HasSuffix(infoLine, " missing") {
		return &ObjectInfo{}, nil
	}

	info := strings.Split(infoLine, " ")

	objectSizeStr := info[2]
	objectSize, err := strconv.ParseInt(objectSizeStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parse object size: %v", err)
	}

	return &ObjectInfo{
		Oid:  info[0],
		Type: info[1],
		Size: objectSize,
	}, nil
}
