package listen

import (
	"context"
	"net"
	"syscall"

	"golang.org/x/sys/windows"
)

var listenConfig = net.ListenConfig{
	Control: func(network, address string, c syscall.RawConn) (err error) {
		return c.Control(func(fd uintptr) {
			var fh = windows.Handle(fd)
			if err = syscall.SetsockoptInt(syscall.Handle(fh), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1); err == nil {
				if err = syscall.SetsockoptInt(syscall.Handle(fh), syscall.IPPROTO_TCP, syscall.TCP_NODELAY, 1); err == nil {
					if err = syscall.SetNonblock(syscall.Handle(fh), true); err == nil {
						if err = syscall.SetsockoptInt(syscall.Handle(fh), syscall.SOL_SOCKET, syscall.SO_RCVBUF, 32768*2); err == nil {
							err = syscall.SetsockoptInt(syscall.Handle(fh), syscall.SOL_SOCKET, syscall.SO_SNDBUF, 32768*2)
						}
					}
				}
			}
		})
	},
}

// Listen returns TCP listener with SO_REUSEADDR option set, SO_REUSEPORT is not supported on Windows, so it uses
// SO_REUSEADDR as an alternative to achieve the same effect.
func Listen(network, addr string) (ln net.Listener, err error) {
	ln, err = listenConfig.Listen(context.Background(), network, addr)
	return
}
