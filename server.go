package mynet

import (
	"fmt"
	"net"
	"os"
	"strings"
	"syscall"
	"time"

	"mynet/poll"
)

/*
参考evio lite代码：
- 移除多地址 1
- 增加udp，unixSocket的实现
- 易用
- demo
- 单测 1
- poll实现，增加接口来封装实现 1

*/

type Server struct {
	lastTick time.Time
	delay    time.Duration
	p        poll.IPoll
	events   Events

	Addr net.Addr
	Ln   net.Listener
	Lf   *os.File // listener's file
	Lfd  int      // listener's fd
}

var continueErr = fmt.Errorf("continue")

func (s *Server) Serve() error {
	p := s.p
	events := s.events
	conns := make(map[int]*tcpConn) // todo Conn
	defer func() {
		s.Close(conns)
	}()

	if events.Serving != nil {
		if events.Serving(s) == Shutdown {
			return nil
		}
	}
	var lastTick time.Time
	s.lastTick = lastTick
	var delay time.Duration = -1
	if events.Tick != nil {
		delay = 0
	}
	s.delay = delay
	packet := make([]byte, 4096)
	var shutdown bool
	for !shutdown {
		fds := p.Wait(s.delay)
	nextfd:
		for _, fd := range fds {
			if fd == s.Lfd { // listenFd accept
				err := s.handleAccept(p, conns, events)
				if err != nil {
					if err == continueErr {
						continue nextfd
					}
					return err
				}
			}
			// 非 listenFd 走这里
			c := conns[fd]
			if len(c.out)-c.outIdx > 0 {
				s.handleWrite(c, p)
			} else if c.action >= Close { // handle close
				c.poll = nil
				syscall.Close(c.fd)
				delete(conns, c.fd)
				if events.Closed != nil {
					action := events.Closed(c)
					if c.action == Shutdown || action == Shutdown {
						shutdown = true
						break
					}
				}
			} else {
				err := s.handleRead(c, packet, events, p, fd)
				if err == continueErr {
					continue
				}
			}
		}
		// 处理完fd事件后，处理定时事件
		if events.Tick != nil {
			done := s.handleTick()
			if done {
				return nil
			}

		}
	}
	return nil
}

func (s *Server) handleAccept(p poll.IPoll, conns map[int]*tcpConn, events Events) error {
	fd, sa, err := syscall.Accept(s.Lfd)
	if err != nil {
		if err == syscall.EAGAIN {
			//continue nextfd
			return continueErr
		}
		return fmt.Errorf("accept err:%v", err)
	}
	if _, ok := s.Ln.(*net.TCPListener); ok {
		if err := poll.SetKeepAlive(fd, 300); err != nil {
			syscall.Close(fd)
			//continue nextfd
			return continueErr
		}
	}
	if err := syscall.SetNonblock(fd, true); err != nil {
		syscall.Close(fd)
		//continue nextfd
		return continueErr
	}
	p.AddRead(fd)
	c := &tcpConn{
		fd:    fd,
		poll:  p,
		laddr: s.Ln.Addr(),
		sa:    sa,
	}
	conns[c.fd] = c
	if events.Opened != nil {
		out, action := events.Opened(c)
		if len(out) > 0 || action != None {
			c.out = append(c.out, out...)
			c.action = action
			c.write = true
			p.ModReadWrite(fd)
		}
	}
	//continue nextfd
	return continueErr
}

func (s *Server) handleRead(c *tcpConn, packet []byte, events Events, p poll.IPoll, fd int) error {
	// 从fd读数据
	n, err := syscall.Read(c.fd, packet[:])
	if err != nil || n == 0 {
		if err != syscall.EAGAIN {
			c.action = Close
		}
		return continueErr
	}
	if events.Data != nil {
		out, action := events.Data(c, packet[:n])
		if len(out) > 0 || action != None {
			c.out = append(c.out, out...)
			c.action = action
			c.write = true
			p.ModReadWrite(fd)
		}
	}
	return nil
}

func (s *Server) handleWrite(c *tcpConn, p poll.IPoll) {
	for {
		n, err := syscall.Write(c.fd, c.out[c.outIdx:])
		if err != nil {
			if err != syscall.EAGAIN {
				if c.action < Close {
					c.action = Close
				}
				break
			}
		}
		c.outIdx += n
		if c.outIdx < len(c.out) {
			continue
		}
		break
	}
	c.outIdx = 0
	if cap(c.out) > 4096 {
		c.out = nil
	} else {
		c.out = c.out[:0]
	}
	if c.action == None {
		c.write = false
		p.ModRead(c.fd)
	}
}

func (s *Server) handleTick() bool {
	now := time.Now()
	if now.Sub(s.lastTick) > s.delay {
		var action Action
		s.lastTick = now
		s.delay, action = s.events.Tick(now)
		if s.delay < 0 {
			s.delay = 0
		}
		if action == Shutdown {
			return true
		}
	}
	return false
}

func NewServer(addr string, events Events) (*Server, error) {
	s := &Server{events: events}
	p := poll.NewPoll()
	s.p = p

	// todo 处理其他协议 unixDomain/udp
	network := "tcp"
	split := "://"
	if strings.Contains(addr, split) {
		network = strings.Split(addr, split)[0]
		addr = strings.Split(addr, split)[1]
	}
	ln, err := net.Listen(network, addr)
	if err != nil {
		return nil, err
	}
	var lnf *os.File
	switch netln := ln.(type) {
	case *net.TCPListener:
		lnf, err = netln.File()
	}
	if err != nil {
		ln.Close()
		return nil, err
	}
	lfd := int(lnf.Fd())
	s.Lfd = lfd
	s.Ln = ln
	s.Lf = lnf
	err = syscall.SetNonblock(lfd, true)
	if err != nil {
		return nil, err
	}
	p.AddRead(lfd)
	return s, nil
}

func (s *Server) Close(conns map[int]*tcpConn) {
	for cfd, c := range conns {
		c.poll = nil
		syscall.Close(cfd)
		if s.events.Closed != nil {
			s.events.Closed(c)
		}
	}

	syscall.Close(s.Lfd)
	s.Lf.Close()
	s.Ln.Close()
}
