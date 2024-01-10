package listen

import (
	"context"
	"net"
	"syscall"

	"golang.org/x/sys/unix"
)

var listenConfig = net.ListenConfig{
	Control: func(network, address string, c syscall.RawConn) (err error) {
		return c.Control(func(fd uintptr) {
			if err = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEADDR, 1); err == nil {
				//if err = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEPORT, 1); err == nil {
				if err = syscall.SetsockoptInt(int(fd), syscall.IPPROTO_TCP, syscall.TCP_NODELAY, 1); err == nil {
					if err = syscall.SetNonblock(int(fd), true); err == nil {
						if err = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_RCVBUF, 32768*2); err == nil {
							err = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_SNDBUF, 32768*2)
						}
					}
				}
				//}
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
