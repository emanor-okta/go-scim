package log

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

var inFlight map[string]Message
var messages []Message

var rwLock sync.RWMutex

type Message struct {
	TimeStamp            time.Time
	TimeStampResp        time.Time
	Method               string
	Response             int
	ResponseStatusString string
	Url                  string
	Headers              string
	RequestBody          string
	ResponseBody         string
}

func Init() {
	//TODO get message history length - for now grows until a crash?
	messages = make([]Message, 0)
	inFlight = make(map[string]Message)
}

func (m Message) FormatDate() string {
	// return m.TimeStamp.Format("Mon Jan 2 15:04:05.999")
	return m.TimeStamp.Format("01/02/2006 15:04:05.999")
}

func (m Message) FormatMessage() string {
	return fmt.Sprintf("------- Headers -------\n%s\n----- Request Body -----\n%s\n----- Response Body -----\n%s", m.Headers, m.RequestBody, m.ResponseBody)
}

func (m Message) FormatElapsedTime() string {
	return fmt.Sprintf("%.4fms", (float64(m.TimeStampResp.UnixMilli()-m.TimeStamp.UnixMilli()))/1000.0)
}

func AddRequest(k string, m Message) {
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
		m.TimeStampResp = time.Now()
		messages = append(messages, m)
		delete(inFlight, k)
	}
	rwLock.Unlock()
}

func AddResponseStatus(k string, status int) {
	rwLock.Lock()
	if m, ok := inFlight[k]; ok {
		m.Response = status
		m.TimeStampResp = time.Now()
		m.ResponseStatusString = http.StatusText(status)
		inFlight[k] = m
	}
	rwLock.Unlock()
}

func GetInFlightMessages() map[string]Message {

	return inFlight
}

func GetMessages(start, count int) ([]Message, int) {
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
