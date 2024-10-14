# syntax=docker/dockerfile:1
#docker build --tag docker-go-scim --no-cache go-scim
#docker image tag docker-go-scim:latest docker-go-scim:v1.00
#docker tag docker-go-scim:v1.00 erikmanoroktacom/docker-go-scim:v1.00
#docker push erikmanoroktacom/docker-go-scim:v1.00

#docker run --publish 8082:8082 docker-go-scim

FROM golang:1.23

# Set destination for COPY
WORKDIR /app

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download
#RUN go get github.com/emanor-okta/go-scim/filters

# Copy the source code. Note the slash at the end, as explained in
# https://docs.docker.com/engine/reference/builder/#copy
COPY . ./

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o /docker-go-scim

# Optional:
# To bind to a TCP port, runtime parameters must be supplied to the docker command.
# But we can document in the Dockerfile what ports
# the application is going to listen on by default.
# https://docs.docker.com/engine/reference/builder/#expose
EXPOSE 8082 9002

# Run
CMD ["/docker-go-scim"]
