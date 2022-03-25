## Schema Independent Scim Server Designed to help Troubleshoot Okta SCIM Integrations

### Prerequisites

Before running this sample, you will need the following:

* An Okta Developer Account, you can sign up for one at https://developer.okta.com/signup/.
* A running [Redis](https://redis.io/) instance.
  * instructions for setting up and running a dockerized instance below
* [Go](https://golang.org/) installed, *1.16+* 
* [ngrok](https://ngrok.com/download)

### To Install
```
git clone https://github.com/emanor-okta/go-scim.git
cd go-scim
go mod tidy
```

### Start Redis in a local [Docker](https://docs.docker.com/get-docker/) Container
```
docker run --name go-scim-redis -p 6379:6379 -d redis redis-server --save 60 1 --loglevel warning
```

### Start go-scim
```
go run main.go
```

### Start an ngrok tunnel
```
ngrok http 8082
```
Grab the ngrok https forwarding address (ie `https://d00a-2601-644-8f82-5ca0-dc67-26b5-abf0-9842.ngrok.io` -> ht<span>tp://localhost:8082).  
When creating a SCIM provisioning integration in Okta this URL will be used with `/scim/v2` appended to it.   
`https://d00a-2601-644-8f82-5ca0-dc67-26b5-abf0-9842.ngrok.io/scim/v2`
  
### Configure Message Debugging
`config.yaml` contains 3 flags to enable message debugging
* debug_headers - log HTTP headers
* debug_body - log POST/PUT message body
* debug_query - log URL query params
  
### Filtering
Many messages can be filtered before persistence and/or filtered before the response.   
* The [default](https://github.com/emanor-okta/go-scim/blob/main/filters/defaultFilter.go) filter is no-op 
* A [sampleFilter](https://github.com/emanor-okta/go-scim/blob/main/filters/sampleFilter.go) has been provided.  

To set the filter, edit [server.go](https://github.com/emanor-okta/go-scim/blob/main/server/server.go) and change `reqFilter = filters.DefaultFilter{}`
  
### Redis CLI
To connect to the redis-cli in the Docker Container
```
docker exec -i go-scim-redis bash
/usr/local/bin/redis-cli
```
Redis commands can be found at https://redis.io/commands.  
To reset the Redis DB use `flushdb`   
  
## Issues
* does not support using special characters in usernames, group names, etc
* no authentication, but printing headers will show what is sent from Okta
  

