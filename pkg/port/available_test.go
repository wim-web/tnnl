package port

import (
	"fmt"
	"net"
	"runtime"
	"runtime/debug"
	"testing"
)

func TestAvailablePortIsReleasedBeforeReturn(t *testing.T) {
	runtime.GC()
	previousGCPercent := debug.SetGCPercent(-1)
	t.Cleanup(func() {
		debug.SetGCPercent(previousGCPercent)
	})

	port, err := AvailablePort()
	if err != nil {
		t.Fatalf("AvailablePort() error = %v", err)
	}
	if port < 1 || port > 65535 {
		t.Fatalf("AvailablePort() = %d, want a valid port", port)
	}

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		t.Fatalf("net.Listen() on wildcard address error = %v", err)
	}
	if err := l.Close(); err != nil {
		t.Fatalf("wildcard listener.Close() error = %v", err)
	}

	l, err = net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatalf("net.Listen() on loopback address error = %v", err)
	}
	if err := l.Close(); err != nil {
		t.Fatalf("loopback listener.Close() error = %v", err)
	}
}
