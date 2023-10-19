package mynet

type Action int

const (
	None Action = iota
	Close
	Shutdown
)
