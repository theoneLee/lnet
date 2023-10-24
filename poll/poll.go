package poll

import (
	"time"
)

// IPoll 多路复用接口
type IPoll interface {
	AddRead(fd int)
	ModRead(fd int)
	ModReadWrite(fd int)

	Wait(timeout time.Duration) []int
}
