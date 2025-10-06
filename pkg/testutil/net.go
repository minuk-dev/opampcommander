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

	var tcpAddr *net.TCPAddr

	tcpAddr, err = net.ResolveTCPAddr("tcp", "localhost:0")
	if err == nil {
		var listener *net.TCPListener

		listener, err = net.ListenTCP("tcp", tcpAddr)
		if err == nil {
			defer func() {
				_ = listener.Close()
			}()

			tcpAddr, ok := listener.Addr().(*net.TCPAddr)
			if !ok {
				return 0, net.InvalidAddrError("not a TCP address")
			}

			return tcpAddr.Port, nil
		}
	}

	return 0, err
}
