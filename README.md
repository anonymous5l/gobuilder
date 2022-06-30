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
* support remote deploy program

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
        deploy: '127.0.0.1:2030' # remote gobuilder-server
        clean-after-deploy: true # after remote deploy remove local binary file
version: 1.18.3 # expect golang version
parallel: 5 # build how many project in once
auto-upgrade: true # auto increment version.patch
ca: gobuilder-root.pem # remote deploy only cert ca
cert: gobuilder-client.pem # remote deploy only client cert
key: gobuilder-client.key # remote deploy only client key
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

## Remote deploy

### Build

```bash
$: go build && mkdir bin && ./gobuilder
```

### Usage

generate root ca, server cert & key, client cert & key

```bash
$: ./gobuilder-server keygen
```

server use `QUIC` protocol base on `UDP` fast and safe

create `server.yaml`

```bash
address: '<IPAddress>:2030'
ca: gobuilder-root.pem # root ca pem
cert: gobuilder-server.pem # server cert pem
key: gobuilder-server.key # server rsa 2048 key
handler: 128 # max handle in same time use ants goroutine library

packages:
  hello-world:
    before-action: /root/gobuilder/gobuilder-before.sh # running before command
    perm: 0755 # default 0755
    executable: /root/gobuilder/hello-world
    after-action: /root/gobuilder/gobuilder-after.sh # running after update command 
  
  # ...
```

running gobuilder deploy server

```bash
$: ./gobuilder-server
Golang build tool server side
```

if modify config use `kill -USR2 <PID>` to reload config `packages` section
