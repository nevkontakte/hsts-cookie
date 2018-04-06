.PHONY: build run

all: build

build: generate
	gb build

generate: src/github.com/nevkontakte/hsts-cookie/webui/assets.go

src/github.com/nevkontakte/hsts-cookie/webui/assets.go: src/github.com/nevkontakte/hsts-cookie/public/index.html
	gb generate github.com/nevkontakte/hsts-cookie/webui

run: build
	sudo ./bin/server -domain hsts.n37.link

docker-build:
	docker build -t nevkontakte/hsts-cookie .

docker-push: docker-build
	docker push nevkontakte/hsts-cookie:latest

docker-run: docker-build
	docker run --rm -it -p 80:80 -p 443:443 -v hsts-cookie-data:/srv \
		--name hsts-cookie nevkontakte/hsts-cookie \
		--domain hsts.n37.link

up: docker-build
	docker stack up  -c docker-compose.yml hsts-cookie

down:
	docker stack down hsts-cookie
