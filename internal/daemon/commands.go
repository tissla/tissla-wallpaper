package daemon

import (
	"log"
	"net"
	"os"
	"tissla-wallpaper/internal/ipc"
)

func (d *Daemon) HandleCommands() {

	sockPath := ipc.SocketPath()
	os.Remove(sockPath)

	ln, err := net.Listen("unix", sockPath)
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}

		buf := make([]byte, 1024)
		n, _ := conn.Read(buf)
		d.handleCommand(string(buf[:n]))
		conn.Close()
	}
}

func (d *Daemon) handleCommand(cmd string) error {

	return nil
}
