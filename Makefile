MAKEFLAGS = -s
export GOPATH := $(shell godep path):$(GOPATH)
PLATFORMS = linux/386 linux/amd64 darwin/amd64

FSEVENTS = build/fsevents-wrapper
TARGET_DIRS = $(wildcard go/cmd/*)
TARGET_GOPKGS = $(addprefix github.com/burke/zeus/,$(TARGET_DIRS))
TARGET_BINARIES = $(addprefix build/,$(TARGET_DIRS:%/=%))

VERSION = go/zeusversion/zeusversion.go

default: build

all: fmt build manpages gem

fmt:
	godep go fmt -x $(dir $(shell find ./go/ -name '*.go'))

manpages:
	cd man; /usr/bin/env rake

gem: $(FSEVENTS) manpages
	mkdir -p rubygem/ext/fsevents-wrapper
	cp -r examples rubygem
	cp build/fsevents-wrapper rubygem/ext/fsevents-wrapper
	cd rubygem; /usr/bin/env rake

$(FSEVENTS): ext/fsevents/*.m
	cd ext/fsevents ; $(MAKE)
	mkdir -p build
	cp ext/fsevents/build/Release/fsevents-wrapper build

build: $(VERSION) $(FSEVENTS)
	gox -osarch="${PLATFORMS}" -output="./build/{{.Dir}}-{{.OS}}-{{.Arch}}" $(TARGET_GOPKGS)

$(VERSION): VERSION
	cd go/zeusversion ; /usr/bin/env ruby ./genversion.rb

install: all
	gem install rubygem/pkg/*.gem --no-ri --no-rdoc

clean:
	rm -f build/*
	cd man; rake clean
	cd ext/fsevents ; $(MAKE) clean
	cd rubygem ; rake clean
	rm -f $(VERSION)

.PHONY: build clean all fmt manpages gem default
