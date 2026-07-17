package port

import (
	"fmt"
	"net"
)

func AvailablePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return -1, fmt.Errorf("select local port: %w", err)
	}

	port := l.Addr().(*net.TCPAddr).Port
	if err := l.Close(); err != nil {
		return -1, fmt.Errorf("release local port %d: %w", port, err)
	}

	return port, nil
}
