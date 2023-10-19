package mynet

import (
	"time"
)

// Events server可以注册的回调事件
type Events struct {
	// Serving server accept()链接时 触发;用于初始化
	Serving func(server Server) (action Action)

	//
	Opened func(c Conn) (out []byte, action Action)

	Closed func(c Conn) (action Action)

	//PreWrite func()

	// Data conn给server发数据时触发
	Data func(c Conn, in []byte) (out []byte, action Action)

	// Tick 在server启动后立即触发，并将在延迟返回值指定的持续时间之后再次触发。
	Tick func(now time.Time) (delay time.Duration, action Action)
}
