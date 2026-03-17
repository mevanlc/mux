package tmux

import "time"

type Session struct {
	Name      string
	Windows   int
	Created   time.Time
	Attached  bool
	Directory string
}
