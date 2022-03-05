IMAGE:="registry.n37.link/hsts-cookie"
TEST_HOST:="hsts.n37.link"

.PHONY: build run

all: build

build:
	go build cmd/server

run: build
	sudo ./server -domain $(TEST_HOST)

docker-build:
	docker build -t $(IMAGE) .

docker-push: docker-build
	docker push $(IMAGE)

docker-run: docker-build
	docker run --rm -it -p 80:80 -p 443:443 -v hsts-cookie-data:/srv \
		--name hsts-cookie $(IMAGE) \
		--domain $(TEST_HOST)

up: docker-build
	docker stack up  -c docker-compose.yml hsts-cookie

down:
	docker stack down hsts-cookie
