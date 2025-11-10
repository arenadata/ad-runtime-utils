package exec

import (
	"net"
	"os"
	"testing"
)

func startSimpleServer(t *testing.T, protocol string) (int, error) {
	// Start a simple TCP server and return the port it's listening on.
	listener, err := net.Listen(protocol, ":0")
	if err != nil {
		return 0, err
	}
	go func() {
		conn, lerr := listener.Accept()
		if lerr != nil {
			t.Error(lerr)
			return
		}
		conn.Close()
	}()
	return listener.Addr().(*net.TCPAddr).Port, nil
}

func TestGetTCPSocketsForPid(t *testing.T) {
	port, err := startSimpleServer(t, string(TCP))
	if err != nil {
		t.Error(err)
	}
	pid := os.Getpid()
	sockets, err := GetTCPSocketsForPid(pid)
	if err != nil {
		t.Error(err)
		return
	}
	for _, soc := range sockets {
		if soc.Port == port {
			return
		}
	}
	t.Error("Expected to find a socket listening on the port of the started server")
}
