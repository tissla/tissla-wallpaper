// Package wallpaperctl
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"
	img "tissla-wallpaper/internal/image"
	"tissla-wallpaper/internal/ipc"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	// parse after verb
	flag.CommandLine.Parse(os.Args[2:])

	cmd, err := handleInput(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	conn, err := net.Dial("unix", ipc.SocketPath())
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	msg, err := json.Marshal(cmd)
	if err != nil {
		log.Fatal(err)
	}
	if _, err = conn.Write(msg); err != nil {
		log.Fatal(err)
	}

	// daemon writes its response then closes the connection
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	resp, err := io.ReadAll(conn)
	if len(resp) > 0 {
		os.Stdout.Write(resp)
	}

	log.Printf("length of response: %d", len(resp))

	if err != nil {
		log.Fatalf("no response: %v", err)
	}
}

func handleInput(cmd string) (ipc.Request, error) {
	switch cmd {
	case "set":
		if *pathPtr == "" {
			return ipc.Request{}, errors.New("set: -path is required")
		}
		if _, err := os.Stat(*pathPtr); err != nil {
			return ipc.Request{}, err
		}
		absPath, err := filepath.Abs(*pathPtr)
		if err != nil {
			return ipc.Request{}, err
		}
		if _, err := img.ParseScaleMode(*modePtr); err != nil {
			return ipc.Request{}, err
		}

		return ipc.Request{
			Verb:   "set",
			Path:   absPath,
			Output: *outputPtr,
			Mode:   *modePtr,
		}, nil
	case "clear":
		return ipc.Request{
			Verb: "clear",
		}, nil
	case "monitors":
		return ipc.Request{
			Verb: "monitors",
		}, nil
	default:
		usage()
		return ipc.Request{}, errors.New("unknown command")
	}
}

func usage() {
	fmt.Println("usage: wallpaperctl set <path> [fill|fit|stretch] | clear | monitors")
}
