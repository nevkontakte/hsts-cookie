.PHONY: build run

all: build

build: generate
	gb build

generate: src/github.com/nevkontakte/hsts-cookie/webui/assets.go

src/github.com/nevkontakte/hsts-cookie/webui/assets.go: src/github.com/nevkontakte/hsts-cookie/public/index.html
	gb generate github.com/nevkontakte/hsts-cookie/webui

run: build
	sudo ./bin/server -domain hsts.n37.link
