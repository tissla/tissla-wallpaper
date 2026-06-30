package main

import (
	"log"
	"tissla-wallpaper/internal/daemon"
)

func main() {
	d, err := daemon.New()

	if err != nil {
		log.Fatal(err)
	}

	go d.HandleCommands()

	log.Printf("Starting tissla-wallpaperd..")
	d.Run()

}
