//usr/bin/env true; exec /usr/bin/env go run "$0" "$@"; exit "$?"
//+build ignore

/*
 * /usr/local/bin/make.go
 * make this file executable (chmod +x make.go)
 *
 * aerth fork: github.com/aerth/make.go
 * original: github.com/nstratos/make.go
 *
 */

// package make.go builds Go projects
package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

type platform struct {
	os   string
	arch string
}

type binary struct {
	name    string
	single  string
	version string
	targets []platform
}

var (
	_release    = "v0.1.1" // make.go release
	cross       = flag.Bool("all", false, "build binaries for all target platforms")
	verbose     = flag.Bool("v", false, "print build output")
	series      = flag.Bool("s", false, "one build at a time")
	cgoenabled  = flag.Bool("cgo", false, "use cgo compiler")
	clean       = flag.Bool("clean", false, "remove all created binaries from current directory")
	buildOS     = flag.String("os", runtime.GOOS, "set operating system to build for")
	buildArch   = flag.String("arch", runtime.GOARCH, "set architecture to build for")
	destination = flag.String("c", "", "change to directory before starting")
	gobin       = flag.String("o", "", "output files to directory, current working directory if blank (see -name)")
	maxproc     = flag.Int("j", 4, "max processes")
	name        = flag.String("name", "", "name file (similar to cc -o)")
	buildmode   = flag.String("buildmode", "default", "see 'go help buildmode'")
	noversion   = flag.Bool("noversion", false, "dont automatically substitute 'version' var")
	rwd         string // real working directory, where binaries will be located
	singlefile  string
)

func init() {
	flag.Usage = usage
	println("make.go " + _release + " [https://github.com/aerth/make.go]")
	log.SetFlags(0)

	// get rwd
	var err error
	rwd, err = os.Getwd()
	if err != nil {
		log.Println(err.Error())
		os.Exit(111)
	}
	// trailing slash
	rwd += "/"
}

func usage() {
	//println("Usage: make.go [OPTIONS] [GO_PACKAGE]\n")
	log.Println("usage:", "make.go", "[OPTIONS]", "[go_project]")
	log.Println("OPTIONS:")
	flag.PrintDefaults()
}

func main() {
	flag.Parse()

	if len(flag.Args()) > 0 && flag.Args()[0] == "install" {
		println("install not supported")
		os.Exit(111)
	}

	// user output directory
	if *gobin != "" {
		rwd = *gobin
	}
	var abs = *destination
	var err error
	// chdir or die
	switch {
	case len(flag.Args()) >= 1 && *destination != "":
		log.Fatalln("\nfatal: use argument or '-c' flag, not both")
	case len(flag.Args()) >= 1 && *destination == "":
		if !strings.HasSuffix(*destination, ".go") {
			*destination = flag.Args()[0]
			if *destination == "." {
				*destination = ""
			}
			if *destination == ".." || *destination == "../" {
				abs, err = filepath.Abs(*destination)
				if err != nil {
					println(err.Error())
					os.Exit(111)
				}
				*destination = abs
			}
			println("using", *destination)

		} else {
			singlefile = flag.Args()[0]
		}
		fallthrough
	case *destination != "":
		abs, err = filepath.Abs(*destination)
		if err != nil {
			println(err.Error())
			os.Exit(111)
		}
		err = os.Chdir(abs)
		if err != nil {
			if strings.Contains(err.Error(), "no such") {
				// go1.8 default $HOME/go
				gopath := os.Getenv("GOPATH")
				if gopath == "" {
					gopath = filepath.Join(os.Getenv("HOME"), "go")
				}
				gopath = filepath.Join(gopath, "src")
				dest := filepath.Join(gopath, *destination)
				err = os.Chdir(dest)

				if err != nil {
					log.Println(err.Error())
					os.Exit(111)
				}
				*destination = dest
			}
		}
	default: // no args, no destination
	}
	abs, err = filepath.Abs(*destination)
	if err != nil {
		println(err.Error())
		os.Exit(111)
	}
	// check if user flags are after arguments, in which case they would have been ignored silently
	for i, fl := range flag.Args() {
		switch {
		case fl == "--":
			continue
		case fl == "-":
			log.Fatalf("\nfatal: bad flag %v: %q", i+1, fl)
		case strings.HasPrefix(fl, "-"):
			log.Fatalln("\nfatal: found flags after arguments")
		}
	}

	// print go version and goroot
	log.Println(runtime.Version()+":", runtime.GOROOT())

	// print name of go compiler
	if *cgoenabled {
		os.Setenv("CGO_ENABLED", "1")
	}
	if "1" == os.Getenv("CGO_ENABLED") {
		log.Println("go compiler: CGO")
	} else {
		os.Setenv("CGO_ENABLED", "0")
		log.Println("compiler: GC")
	}
	log.Println("buildmode:", *buildmode)
	// print max procs
	runtime.GOMAXPROCS(*maxproc)
	println("gomaxprocs:", runtime.GOMAXPROCS(*maxproc))

	// get binary name
	if *destination == "" {
		println("setting destination to:", ".")
		*destination = "."
	}

	println("absolute:", abs)
	bin := binary{
		name: getMainProjectName(filepath.Base(abs)),
		// Change these according to the platforms you would like to support. A
		// full list can be found here:
		// https://golang.org/doc/install/source#environment
		targets: []platform{
			{os: "linux", arch: "386"}, {os: "linux", arch: "amd64"},
			{os: "linux", arch: "arm"},
			{os: "windows", arch: "386"}, {os: "windows", arch: "amd64"},
			{os: "darwin", arch: "386"}, {os: "darwin", arch: "amd64"},
			{os: "openbsd", arch: "386"}, {os: "openbsd", arch: "amd64"},
			{os: "netbsd", arch: "386"}, {os: "netbsd", arch: "amd64"},
			{os: "freebsd", arch: "386"}, {os: "freebsd", arch: "amd64"},
		},
	}

	bin.version = getVersion(abs)
	log.Println("building:", bin.name, bin.version)
	if *series {
		if *cross {
			forEachBinTargetSeries(bin, buildBinary)
			os.Exit(0)
		}

		if *clean {
			forEachBinTargetSeries(bin, rmBinary)
			os.Exit(0)
		}

		log.Println("need -all flag")
		os.Exit(111)
	}

	if *cross {
		forEachBinTargetParallel(bin, buildBinary)
		os.Exit(0)
	}

	if *clean {
		forEachBinTargetParallel(bin, rmBinary)
		os.Exit(0)
	}

	buildBinary(bin, *buildOS, *buildArch)
}

// getVersion returns the version of the project using git.
func getVersion(abs string) string {
	cmd := exec.Command("git", "describe", "--tags", "--always")
	cmd.Dir = abs
	buf := new(bytes.Buffer)
	cmd.Stderr = buf
	out, err := cmd.Output()
	if err != nil {
		return ""
	}

	return strings.TrimPrefix(strings.TrimSpace(string(out)), "v")
}

type binaryFunc func(bin binary, OS, arch string)

// forEachBinTarget runs a function for all the target platforms of a binary in
// parallel.
func forEachBinTargetParallel(bin binary, fn binaryFunc) {
	var wg sync.WaitGroup
	for _, t := range bin.targets {
		wg.Add(1)
		go func(bin binary, os, arch string) {
			defer wg.Done()
			fn(bin, os, arch)
		}(bin, t.os, t.arch)
	}
	wg.Wait()
}
func forEachBinTargetSeries(bin binary, fn binaryFunc) {
	for _, t := range bin.targets {
		func(bin binary, os, arch string) {
			fn(bin, os, arch)
		}(bin, t.os, t.arch)
	}
}

// buildBinary builds a binary for a certain operating system and architecture
// combination while using --ldflags to set the binary's version variable.
func buildBinary(bin binary, OS, arch string) {
	t1 := time.Now()
	ldflags := fmt.Sprintf("--ldflags=-s -w -X main.version=%s", bin.version)
	gcflags := "--gcflags=-trimpath $GOPATH/src"
	if *noversion {
		ldflags = "--ldflags=-s -w"
	}
	cmd := exec.Command("go", "build", gcflags, "-buildmode="+*buildmode, "-x", ldflags, "-o", rwd+bin.Name(OS, arch), singlefile)
	if *name != "" {
		cmd = exec.Command("go", "build", gcflags, "-buildmode="+*buildmode, "-x", ldflags, "-o", filepath.Join(rwd, *name))
	}
	buf := new(bytes.Buffer)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if *verbose {
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		cmd.Stdin = os.Stdin
	}

	cmd.Env = copyOSEnv()
	cmd.Env = setEnv(cmd.Env, "GOOS", OS)
	cmd.Env = setEnv(cmd.Env, "GOARCH", arch)
	if err := cmd.Run(); err != nil {
		if !strings.Contains(buf.String(), "no buildable Go source files") {
			log.Println("\nBUILD ERROR:", err.Error(), buf.String())
		} else {
			log.Println("\nfatal: not a Go project")
			log.Println("usage:", "make.go", "[go_project]")
		}
		os.Exit(111)
	}

	// "Built make.go (1min2sec)"
	log.Printf("built %s (%s)", bin.Name(OS, arch), time.Now().Sub(t1))
}

// rmBinary removes a binary from the current directory.
func rmBinary(bin binary, OS, arch string) {
	err := os.Remove(rwd + bin.Name(OS, arch))
	if err != nil {
		if !os.IsNotExist(err) {
			println("error removing binary:", err)
		}
	}
}

// copyOSEnv returns a copy of the system's environment variables.
func copyOSEnv() (environ []string) {
	return append(environ, os.Environ()...)
}

// setEnv searches in a slice of environment variables with the form key=value
// for a key and if found it sets its value, otherwise it adds the pair.
func setEnv(environ []string, key, value string) []string {
	for i, env := range environ {
		if strings.Split(env, "=")[0] == key {
			environ[i] = fmt.Sprintf("%s=%s", key, value)
			return environ
		}
	}
	return append(environ, fmt.Sprintf("%s=%s", key, value))
}

// Name returns the name of the binary with a certain format for a platform.
func (bin binary) Name(os, arch string) string {

	s := fmt.Sprintf("%s_%s-%s", bin.name, os, arch)

	if bin.version != "" {
		s = fmt.Sprintf("%s_%s_%s-%s", bin.name, bin.version, os, arch)
	}

	if os == "windows" {
		s = s + ".exe"
	}
	return s
}

// Names returns the name of the binary for all the target platforms.
func (bin binary) Names() []string {
	names := make([]string, len(bin.targets))
	for i, t := range bin.targets {
		names[i] = bin.Name(t.os, t.arch)
	}
	return names
}

func getMainProjectName(dir string) string {
	dir = strings.TrimSuffix(dir, "/")
	dirname := strings.Split(dir, "/")
	projectName := dirname[len(dirname)-1]
	return projectName
}
