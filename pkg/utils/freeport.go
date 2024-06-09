package utils

import "net"

func FreePort(preferred int) (port int, err error) {
	if preferred != 0 {
		var l *net.TCPListener
		if l, err = net.ListenTCP("tcp", &net.TCPAddr{Port: preferred}); err == nil {
			defer l.Close()
			return l.Addr().(*net.TCPAddr).Port, nil
		}
	}

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
