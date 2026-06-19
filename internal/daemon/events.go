package daemon

func (d *Daemon) HandleEvents() {
	for msg := range d.wlConn.Listen() {

	}

}
