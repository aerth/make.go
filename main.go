package main

import (
	"fmt"
	"net"
	"os"
	"runtime"
)

var (
	version = "devel"
)

func main() {
	if len(os.Args) > 1 {println("Demo binary (safe to delete). Learn more: [https://github.com/aerth/make.go]");os.Exit(111)}
	fmt.Printf("%s %s (runtime: %s)\n", os.Args[0], version, runtime.Version())
	_ = net.Interfaces
}
