package log

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/emanor-okta/go-scim/utils"
)

var inFlight map[string]Message
var messages []Message

var rwLock sync.RWMutex

type Message struct {
	TimeStamp            time.Time
	Method               string
	Response             int
	ResponseStatusString string
	Url                  string
	Headers              string
	RequestBody          string
	ResponseBody         string
}

func Init(config *utils.Configuration) {
	//TODO get message history length - for now grows until a crash?
	messages = make([]Message, 0)
	inFlight = make(map[string]Message)
}

func (m Message) FormatDate() string {
	return m.TimeStamp.Format("Mon Jan 2 15:04:05.999")
}

func (m Message) FormatMessage() string {
	return fmt.Sprintf("------- Headers -------\n%s\n----- Request Body -----\n%s\n----- Response Body -----\n%s", m.Headers, m.RequestBody, m.ResponseBody)
}

func AddRequest(k string, m Message) {
	// fmt.Printf("Key: %v, Message: %+v\n", k, m)
	// if the request comes from web interface ignore
	if strings.Contains(m.Headers, "Go-http-client/1.1") {
		return
	}
	rwLock.Lock()
	inFlight[k] = m
	rwLock.Unlock()
}

func AddResponse(k string, respBody string) {
	rwLock.Lock()
	if m, ok := inFlight[k]; ok {
		m.ResponseBody = respBody
		messages = append(messages, m)
		delete(inFlight, k)
	}
	rwLock.Unlock()
	// fmt.Println("MESSAGES::")
	// for _, m := range messages {
	// 	fmt.Printf("%+v\n", m)
	// }
}

func AddResponseStatus(k string, status int) {
	rwLock.Lock()
	if m, ok := inFlight[k]; ok {
		// fmt.Printf("Found >>>> %+v\n", m)
		m.Response = status
		m.ResponseStatusString = http.StatusText(status)
		inFlight[k] = m
	}
	rwLock.Unlock()
}

func GetInFlightMessages() map[string]Message {

	return inFlight
}

func GetMessages(start, count int) ([]Message, int) {
	// fmt.Printf("l: %v, [0]: %+v\n", len(messages), messages)
	end := start + count
	rwLock.RLock()
	l := len(messages)
	if l <= start {
		rwLock.RUnlock()
		return []Message{}, l
	}
	if l < end {
		end = l
	}
	messagesCopy := make([]Message, end-start)
	copy(messagesCopy, messages[start:end])
	rwLock.RUnlock()
	return messagesCopy, l
}

func FlushMessages() {
	rwLock.Lock()
	messages = make([]Message, 0)
	inFlight = make(map[string]Message)
	rwLock.Unlock()
}
