// Package wallpaperctl
package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	img "tissla-wallpaper/internal/image"
	"tissla-wallpaper/internal/ipc"
)

func main() {

	if len(os.Args) < 2 {
		os.Exit(1)
	}

	cmd := os.Args[1]

	// command parsing
	msg, err := handleInput(cmd)
	if err != nil {
		log.Fatal(err)
	}
	if msg == "" {
		os.Exit(1)
	}

	// write to socket
	sPath := ipc.SocketPath()

	conn, err := net.Dial("unix", sPath)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	conn.Write([]byte(msg))
}

func handleInput(cmd string) (string, error) {
	switch cmd {
	case "set":
		if len(os.Args) < 3 {
			return "", errors.New("set: missing path")
		}
		path := os.Args[2]
		if _, err := os.Stat(path); err != nil {
			return "", err
		}
		absPath, err := filepath.Abs(path)
		if err != nil {
			return "", err
		}

		mode := "fill"
		if len(os.Args) > 3 {
			if _, err := img.ParseScaleMode(os.Args[3]); err != nil {
				return "", err
			}
			mode = os.Args[3]
		}

		return "set " + absPath + " " + mode, nil
	case "clear":
		return "clear", nil
	case "monitors":
		return "monitors", nil
	default:
		usage()
		return "", errors.New("unknown command")
	}
}

func usage() {
	fmt.Println("how to use")
}
