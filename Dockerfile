FROM golang:1.17-alpine AS go-builder
WORKDIR /app
COPY . .
RUN go build ./cmd/server

FROM alpine:latest
RUN apk add --update ca-certificates
WORKDIR /app
COPY --from=go-builder /app/server /usr/local/bin/hsts-cookie
VOLUME /srv
EXPOSE 80 443
ENTRYPOINT ["hsts-cookie", "-acme_dir=/srv/acme-cache"]

