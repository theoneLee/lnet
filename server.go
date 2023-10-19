package mynet

import (
	"net"
	"os"
	"syscall"
)

/*
参考evio lite代码：
- 移除多地址
- 增加udp，unixSocket的实现
- 易用
- demo
- 单测
- poll实现，增加接口来封装实现

*/

type Server struct {
	Addr net.Addr // todo 移除掉多地址逻辑
	Ln   net.Listener
	Lf   *os.File // listener's file
	Lfd  int      // listener's fd
}

// todo
func (s *Server) Serve(events Events, addr string) error {

	defer func() {
		s.Close()
	}()

	p := poll.NewPoll()

}

func (s *Server) Close() {
	syscall.Close(s.Lfd)
	s.Lf.Close()
	s.Ln.Close()
}
