package daemon

import (
	"log"
	wl "tissla-wallpaper/internal/wayland"
)

// Run contains the daemons main loop, which reads the event- and command-channels respectively.
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
			resp, err := d.handleCommand(cmd)
			if err != nil {
				// log error and move on
				resp = "error: " + err.Error()
				log.Printf("command: %v", err)
			}

			if cmd.reply != nil {
				cmd.reply <- resp
			}
		}
	}
}
