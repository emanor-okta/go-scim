#!/bin/bash

# ex ./build.sh 1.22

cd ..

echo "Building go-scim for docker, version v$1"
docker build --tag docker-go-scim --no-cache go-scim

echo "Tagging"
docker image tag docker-go-scim:latest docker-go-scim:v$1
docker tag docker-go-scim:v$1 erikmanoroktacom/docker-go-scim:v$1

echo "Pushing to Docker Hub"
docker push erikmanoroktacom/docker-go-scim:v$1