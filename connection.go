package smtp

import (
	"net"
	"net/textproto"
	"strings"
)

func newConn(conn net.Conn) connection {
	return connection{textproto.NewConn(conn)}
}

type connection struct {
	*textproto.Conn
}

func (conn connection) read() (string, string, error) {
	line, err := conn.ReadLine()
	if err != nil {
		return "", "", err
	}

	parts := strings.SplitN(line, " ", 2)
	if len(parts) == 1 {
		return parts[0], "", nil
	}

	return parts[0], parts[1], nil
}

func (conn connection) readAll() ([]byte, error) {
	d, err := conn.ReadDotBytes()
	if err != nil {
		return []byte{}, err
	}

	return d, nil
}

func (conn connection) write(format string, args ...interface{}) {
	conn.PrintfLine(format, args...)
}
