package wayland

import (
	"os"
	"syscall"
	"unsafe"
)

const sysMemfdCreate = 319 // x86_64 linux
type ShmBuffer struct {
	Fd   int
	Data []byte
}

func NewShmBuffer(size int) (*ShmBuffer, error) {

	fd, err := memfdCreate("wallpaper")
	if err != nil {
		return nil, err
	}

	if err = syscall.Ftruncate(fd, int64(size)); err != nil {
		syscall.Close(fd)
		return nil, err
	}

	data, err := syscall.Mmap(
		fd, 0, size,
		syscall.PROT_READ|syscall.PROT_WRITE,
		syscall.MAP_SHARED,
	)
	if err != nil {
		syscall.Close(fd)
		return nil, err
	}

	return &ShmBuffer{
		Fd:   fd,
		Data: data,
	}, nil
}

// memfdCreate makes the system call memfd_create
// it returns a file descriptor that refers to the anonymous file created
// this file lives in RAM, and will be automatically released when all references to it are dropped
func memfdCreate(name string) (int, error) {
	namePtr, err := syscall.BytePtrFromString(name)
	if err != nil {
		return 0, err
	}
	fd, _, errno := syscall.Syscall(
		sysMemfdCreate,
		uintptr(unsafe.Pointer(namePtr)),
		0,
		0,
	)
	if errno != 0 {
		return 0, os.NewSyscallError("memfd_create", errno)
	}
	return int(fd), nil
}
