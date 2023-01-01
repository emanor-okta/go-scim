package log

import (
	"time"

	"github.com/emanor-okta/go-scim/utils"
)

var inFlight map[string]Message
var messages []Message

type Message struct {
	TimeStamp    time.Time
	Method       string
	Response     int
	Url          string
	Headers      string
	RequestBody  string
	ResponseBody string
}

func Init(config *utils.Configuration) {
	//TODO get message history length - for now grows until a crash?
	messages = make([]Message, 0)
	inFlight = make(map[string]Message)
}

func AddRequest(k string, m Message) {
	// fmt.Printf("Key: %v, Message: %+v\n", k, m)
	inFlight[k] = m
}

func AddResponse(k string, respBody string) {
	if m, ok := inFlight[k]; ok {
		m.ResponseBody = respBody
		messages = append(messages, m)
		delete(inFlight, k)
	}
	// fmt.Println("MESSAGES::")
	// for _, m := range messages {
	// 	fmt.Printf("%+v\n", m)
	// }
}
