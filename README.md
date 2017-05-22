# [make.go](https://github.com/aerth/make.go/archive/master.zip)


## Features

- Automated versioning of produced binaries if project is a git repo
- Parallel cross compilation for all target platforms. (-all flag)
- `gc` by default, use -cgo flag or CGO_ENABLED=1 var to use `cgo`
- Produces stripped, static linked binaries for all platforms in one easy command: 'make.go -v -all'
- Creates smaller binaries than a simple 'go build': 636704 vs 960134
## Usage

`make.go` should be made executable (`chmod +x`).

It can be installed system-wide (`/usr/local/bin/make.go`), or customized and packaged with your project's source code.

- `make.go` builds a versioned binary for the current platform
- `make.go -all` builds versioned binaries for all target platforms.
- `make.go -clean` removes produced binaries (removes current releases only).
- `make.go -os windows -arch amd64` builds a versioned binary for a specific platform.
- `make.go -c path/to/project` changes to directory before building. binaries will end up in $PWD
- `make.go path/to/project` changes to directory before building. binaries will end up in $PWD

## Good alternatives

- A Makefile or other script.
- [Gomaker](https://github.com/aerth/gomaker), Makefile generator for Go projects

## License

The source code is public domain.

## Original make.go

https://github.com/nstratos/make.go
