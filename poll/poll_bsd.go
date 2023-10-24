//go:build darwin || netbsd || freebsd || openbsd || dragonfly
// +build darwin netbsd freebsd openbsd dragonfly

package poll

import (
	"syscall"
	"time"
)

type Poll struct {
	fd      int
	changes []syscall.Kevent_t
	events  []syscall.Kevent_t
	evfds   []int
}

func NewPoll() IPoll {
	fd, err := syscall.Kqueue()
	if err != nil {
		panic(err)
	}
	p := &Poll{
		fd:      fd,
		changes: make([]syscall.Kevent_t, 0),
		events:  make([]syscall.Kevent_t, 64),
		evfds:   make([]int, 0),
	}
	return p
}

func (p *Poll) AddRead(fd int) {
	p.changes = append(p.changes, syscall.Kevent_t{
		Ident:  uint64(fd),
		Filter: syscall.EVFILT_READ,
		Flags:  syscall.EV_ADD,
		//Fflags: 0,
		//Data:   0,
		//Udata:  nil,
	})
}

func (p *Poll) ModRead(fd int) {
	p.changes = append(p.changes, syscall.Kevent_t{
		Ident:  uint64(fd),
		Filter: syscall.EVFILT_WRITE,
		Flags:  syscall.EV_DELETE,
		//Fflags: 0,
		//Data:   0,
		//Udata:  nil,
	})
}

func (p *Poll) ModReadWrite(fd int) {
	p.changes = append(p.changes, syscall.Kevent_t{
		Ident:  uint64(fd),
		Filter: syscall.EVFILT_WRITE,
		Flags:  syscall.EV_ADD,
		//Fflags: 0,
		//Data:   0,
		//Udata:  nil,
	})
}

func (p *Poll) Wait(timeout time.Duration) []int {
	var n int
	var err error
	if timeout >= 0 {
		var ts syscall.Timespec
		ts.Nsec = int64(timeout)
		n, err = syscall.Kevent(p.fd, p.changes, p.events, &ts)
	} else {
		n, err = syscall.Kevent(p.fd, p.changes, p.events, nil)
	}
	if err != nil && err != syscall.EINTR {
		panic(err)
	}
	p.changes = p.changes[:0]
	p.evfds = p.evfds[:0]
	for i := 0; i < n; i++ {
		p.evfds = append(p.evfds, int(p.events[i].Ident))
	}
	return p.evfds
}

func SetKeepAlive(fd, secs int) error {
	return nil
}
