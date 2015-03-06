#!/bin/bash
set -e
cd $(dirname $0)/..
docker build -t hsts .
docker stop "hsts_server" || true
docker rm "hsts_server" || true
docker create -p 80:8080 -p 443:4343 --name "hsts_server" hsts
docker start hsts_server