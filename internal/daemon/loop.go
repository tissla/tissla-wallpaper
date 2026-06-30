package daemon

import (
	"log"
	wl "tissla-wallpaper/internal/wayland"
)

// Run() contains the daemons main loop, which reads the event- and command-channels respectively.
func (d *Daemon) Run() {
	events := d.wlConn.Listen()
	for {
		select {
		case raw, ok := <-events:
			if !ok {
				return
			}
			d.handleEvent(wl.Message(raw))
		case cmd := <-d.commands:
			if err := d.handleCommand(cmd); err != nil {
				// log error and move on
				log.Printf("command: %v", err)
			}
		}
	}
}
