package log

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

const proxy_msg = "proxy.gohtml"
const scim_msg = "messages.gohtml"

var inFlight map[string]Message
var messages []Message
var proxyMessages []Message

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
	ResponseHeaders      string
}

func Init() {
	//TODO get message history length - for now grows until a crash?
	messages = make([]Message, 0)
	proxyMessages = make([]Message, 0)
	inFlight = make(map[string]Message)
}

func (m Message) FormatDate() string {
	return m.TimeStamp.Format("Jan 2 15:04:05.000")
	//return m.TimeStamp.Format("01/02/2006 15:04:05.999")
}

func (m Message) FormatMessage() string {
	if m.ResponseHeaders != "" {
		return fmt.Sprintf("------- Request Headers -------\n%s\n----- Request Body -----\n%s\n------- Response Headers -------\n%s\n----- Response Body -----\n%s", m.Headers, m.RequestBody, m.ResponseHeaders, m.ResponseBody)
	} else {
		return fmt.Sprintf("------- Request Headers -------\n%s\n----- Request Body -----\n%s\n----- Response Body -----\n%s", m.Headers, m.RequestBody, m.ResponseBody)
	}
}

func (m Message) FormatElapsedTime() string {
	return fmt.Sprintf("%.4fms", (float64(m.TimeStampResp.UnixMilli()-m.TimeStamp.UnixMilli()))/1000.0)
}

func AddRequest(k string, m Message) {
	// if the request comes from web interface ignore
	// fmt.Printf("Key: %s Adding: %+v\n", k, m)
	if strings.Contains(m.Headers, "Go-http-client/1.1") {
		return
	}
	rwLock.Lock()
	inFlight[k] = m
	rwLock.Unlock()
}

func AddResponse(k, respBody, msgType string, respHeader *string) {
	rwLock.Lock()
	// fmt.Printf("Add Response key: %s, type: %s\n", k, msgType)
	if m, ok := inFlight[k]; ok {
		// fmt.Printf("found: %v\n", ok)
		m.ResponseBody = respBody
		m.TimeStampResp = time.Now()
		if respHeader != nil {
			m.ResponseHeaders = *respHeader
		}

		if scim_msg == msgType {
			messages = append(messages, m)
		} else {
			proxyMessages = append(proxyMessages, m)
		}
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

func GetMessages(start, count int, msgType string) ([]Message, int) {
	var msgs *[]Message
	if msgType == proxy_msg {
		msgs = &proxyMessages
	} else {
		msgs = &messages
	}
	end := start + count
	rwLock.RLock()
	l := len(*msgs)
	if l <= start {
		rwLock.RUnlock()
		return []Message{}, l
	}
	if l < end {
		end = l
	}
	messagesCopy := make([]Message, end-start)
	copy(messagesCopy, (*msgs)[start:end])
	rwLock.RUnlock()
	return messagesCopy, l
}

func FlushMessages() {
	rwLock.Lock()
	messages = make([]Message, 0)
	proxyMessages = make([]Message, 0)
	inFlight = make(map[string]Message)
	rwLock.Unlock()
}
