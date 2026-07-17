package port

import (
	"errors"
	"fmt"
	"net"
	"testing"
)

type fakeListener struct {
	addr     *net.TCPAddr
	closeErr error
	closed   bool
}

func (*fakeListener) Accept() (net.Conn, error) {
	panic("unexpected Accept call")
}

func (l *fakeListener) Close() error {
	l.closed = true
	return l.closeErr
}

func (l *fakeListener) Addr() net.Addr {
	return l.addr
}

func TestAvailablePortClosesListenerBeforeReturningSelectedPort(t *testing.T) {
	const wantPort = 43210
	l := &fakeListener{addr: &net.TCPAddr{Port: wantPort}}
	var gotNetwork string
	var gotAddress string

	port, err := availablePort(func(network, address string) (net.Listener, error) {
		gotNetwork = network
		gotAddress = address
		return l, nil
	})
	if err != nil {
		t.Fatalf("availablePort() error = %v", err)
	}
	if gotNetwork != "tcp" {
		t.Errorf("listen network = %q, want %q", gotNetwork, "tcp")
	}
	if gotAddress != "127.0.0.1:0" {
		t.Errorf("listen address = %q, want %q", gotAddress, "127.0.0.1:0")
	}
	if port != wantPort {
		t.Errorf("availablePort() port = %d, want %d", port, wantPort)
	}
	if !l.closed {
		t.Error("availablePort() returned before closing listener")
	}
}

func TestAvailablePortPreservesListenError(t *testing.T) {
	wantErr := errors.New("listen failed")

	port, err := availablePort(func(string, string) (net.Listener, error) {
		return nil, wantErr
	})
	if port != -1 {
		t.Errorf("availablePort() port = %d, want -1", port)
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("availablePort() error = %v, want error wrapping %v", err, wantErr)
	}
}

func TestAvailablePortPreservesCloseError(t *testing.T) {
	wantErr := errors.New("close failed")
	l := &fakeListener{
		addr:     &net.TCPAddr{Port: 43210},
		closeErr: wantErr,
	}

	port, err := availablePort(func(string, string) (net.Listener, error) {
		return l, nil
	})
	if port != -1 {
		t.Errorf("availablePort() port = %d, want -1", port)
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("availablePort() error = %v, want error wrapping %v", err, wantErr)
	}
	if !l.closed {
		t.Error("availablePort() did not close listener")
	}
}

func TestAvailablePortIsReleasedBeforeReturn(t *testing.T) {
	port, err := AvailablePort()
	if err != nil {
		t.Fatalf("AvailablePort() error = %v", err)
	}
	if port < 1 || port > 65535 {
		t.Fatalf("AvailablePort() = %d, want a valid port", port)
	}

	l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatalf("net.Listen() on loopback address error = %v", err)
	}
	if err := l.Close(); err != nil {
		t.Fatalf("loopback listener.Close() error = %v", err)
	}
}
