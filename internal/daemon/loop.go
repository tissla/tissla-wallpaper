package daemon

import (
	"log"
	wl "tissla-wallpaper/internal/wayland"
)

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
				log.Printf("command: %v", err)
			}
		}
	}
}
