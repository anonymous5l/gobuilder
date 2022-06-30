# Go build tool

## Introduce

```NOTICE: require go version at least 1.18.x```

project for `Apple Silicon` machine. 
build `x86/x64` program on `Linux` or `Windows`.

## Features

* support batch build
* support docker build
* support host build
* support insert custom `git` variable to program
* version control auto upgrade `patch`
* support remote deploy program protocol use QUIC

## TODO
* remote build
* before build run `go test`
* ...

## Usage

create `.gobuild` file to project dir.

```yaml
packages:
    hello-world: # project name output binary name
        package: gobuilder/cli # go module package like `go build xxx/xxx` 
        # `git` `version` `build` variable info store location
        # internal variable 
        # Version    string
        # BuildStamp string
        # BuildTool  string
        # GitHash    string
        verbose-package: gobuilder/cli/env
        build-flag: [] # custom `go build` suffix
        build-mode: docker # host or docker
        build-os: linux # binary target os
        build-arch: amd64 # binary target arch
        version: # binary version
            major: 1
            minor: 1
            patch: 2 # if `auto-upgrade` == true patch auto increment each build
        dest: bin # binary output directory
version: 1.18.3 # expect golang version
parallel: 5 # build how many project in once
auto-upgrade: true # auto increment version.patch
```

put code in `.gobuilder` then

```bash
go get -u github.com/anonymous5l/gobuilder
```

```bash
$: cd project-dir 
$: gobuilder
$: gobuilder hello-world
```
