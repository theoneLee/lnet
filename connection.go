package mynet

import (
	"net"
	"syscall"
)

type Conn interface {
	Context() any
	SetContext(any)

	// AddrIndex() int

	LocalAddr() net.Addr
	RemoteAddr() net.Addr

	Write(data []byte)
	Close()
}

type tcpConn struct {
	write  bool
	fd     int
	outIdx int    // output write index
	out    []byte // output buffer
	action Action
	ctx    any
	poll   *poll.Poll
	raddr  net.Addr
	laddr  net.Addr
	sa     syscall.Sockaddr // fd's socket address
}

func (t *tcpConn) Context() any {
	return t.ctx
}

func (t *tcpConn) SetContext(ctx any) {
	t.ctx = ctx
}

func (t *tcpConn) LocalAddr() net.Addr {
	return t.laddr
}

func (t *tcpConn) RemoteAddr() net.Addr {
	if t.raddr == nil {
		switch sa := t.sa.(type) {
		case *syscall.SockaddrInet4:
			t.raddr = &net.TCPAddr{
				IP:   append([]byte{}, sa.Addr[:]...),
				Port: sa.Port,
			}
		case *syscall.SockaddrInet6:
			var zone string
			if sa.ZoneId != 0 {
				ifi, err := net.InterfaceByIndex(int(sa.ZoneId))
				if err == nil {
					zone = ifi.Name
				}
			}
			t.raddr = &net.TCPAddr{
				IP:   append([]byte{}, sa.Addr[:]...),
				Port: sa.Port,
				Zone: zone,
			}
		}
	}
	return t.raddr
}

func (t *tcpConn) Write(data []byte) {
	if t.poll == nil {
		return
	}
	if t.action == None {
		t.out = append(t.out, data...)
		if !t.write {
			t.poll.ModReadWrite(t.fd)
			t.write = true
		}
	}
}

func (t *tcpConn) Close() {
	if t.poll == nil {
		return
	}
	if t.action == None {
		t.action = Close
	}
	if !t.write {
		t.poll.ModReadWrite(t.fd)
		t.write = true
	}
}
