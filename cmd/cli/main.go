// Package wallpaperctl
package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	img "tissla-wallpaper/internal/image"
	"tissla-wallpaper/internal/ipc"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	msg, err := handleInput(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	conn, err := net.Dial("unix", ipc.SocketPath())
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	if _, err = io.WriteString(conn, msg); err != nil {
		log.Fatal(err)
	}

	// daemon writes its response then closes the connection
	resp, err := io.ReadAll(conn)
	if err != nil {
		log.Fatal(err)
	}
	os.Stdout.Write(resp)
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
	fmt.Println("usage: wallpaperctl set <path> [fill|fit|stretch] | clear | monitors")
}
