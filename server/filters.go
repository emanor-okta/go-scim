package server

type ScimRequest interface {
	Request(string) string
}

type ScimResponse interface {
	Response() string
}
