package testutil

import "net"

// GetFreeTCPPort returns a free TCP port that is ready to use.
func (b *Base) GetFreeTCPPort() int {
	b.t.Helper()

	port, err := GetFreeTCPPort()
	if err != nil {
		b.t.Fatalf("failed to get free TCP port: %v", err)
	}

	return port
}

// GetFreeTCPPort asks the kernel for a free open port that is ready to use.
//
//nolint:wrapcheck
func GetFreeTCPPort() (int, error) {
	var err error

	var a *net.TCPAddr

	if a, err = net.ResolveTCPAddr("tcp", "localhost:0"); err == nil {
		var l *net.TCPListener
		if l, err = net.ListenTCP("tcp", a); err == nil {
			defer closeSiliencely(l)

			tcpAddr, ok := l.Addr().(*net.TCPAddr)
			if !ok {
				return 0, net.InvalidAddrError("not a TCP address")
			}
			
return tcpAddr.Port, nil
		}
	}

	return 0, err
}
