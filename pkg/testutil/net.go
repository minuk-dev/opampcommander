package testutil

import "net"

func (b *Base) GetFreeTCPPort() int {
	b.t.Helper()

	port, err := GetFreeTCPPort()
	if err != nil {
		b.t.Fatalf("failed to get free TCP port: %v", err)
	}

	return port
}

// GetFreePort asks the kernel for a free open port that is ready to use.
func GetFreeTCPPort() (port int, err error) {
	var a *net.TCPAddr
	if a, err = net.ResolveTCPAddr("tcp", "localhost:0"); err == nil {
		var l *net.TCPListener
		if l, err = net.ListenTCP("tcp", a); err == nil {
			defer l.Close()
			return l.Addr().(*net.TCPAddr).Port, nil
		}
	}
	return
}
