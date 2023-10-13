package log

import (
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"net/http"

	"github.com/chromedp/cdproto/har"
)

const (
	version      string = "1.2"
	name         string = "Go-Scim har Generator"
	page_ref     string = "page_1"
	http_version string = "HTTP/1.2"
)

func GenerateHar(messages []Message) *har.HAR {
	log := har.Log{}
	log.Version = version
	log.Creator = &har.Creator{
		Name:    name,
		Version: version,
		Comment: "",
	}
	log.Comment = fmt.Sprintf("Har Generated at: %s", time.Now().Format("Jan 2 15:04:05.000"))

	addPages(&log, &messages[0])
	addEntries(&log, messages)

	return &har.HAR{Log: &log}
}

func addPages(log *har.Log, message *Message) {
	/*
	 * A browser would add an entry for each page the user-agent navigates to.
	 * Instead here will create a single Page for all entries to reference
	 */
	log.Pages = []*har.Page{{
		StartedDateTime: message.TimeStamp.Format(time.RFC3339),
		ID:              page_ref,
		Title:           "PlaceHolder",
		PageTimings:     &har.PageTimings{},
		Comment:         "PlaceHolder Page",
	}}
}

func addEntries(log *har.Log, messages []Message) {
	for _, m := range messages {
		e := har.Entry{}
		e.Pageref = page_ref
		e.Cache = &har.Cache{}
		e.StartedDateTime = m.TimeStamp.Format(time.RFC3339)
		e.Time = float64(m.TimeStampResp.UnixMilli() - m.TimeStamp.UnixMilli())
		e.Timings = &har.Timings{
			Send:    0.0,
			Wait:    0.0,
			Receive: 0.0,
		}
		e.Request = addRequest(&m)
		e.Response = addResponse(&m)

		log.Entries = append(log.Entries, &e)
	}
}

func addRequest(m *Message) *har.Request {
	r := har.Request{}
	r.Method = m.Method
	r.URL = m.Url
	r.HTTPVersion = http_version
	r.QueryString = getQueryString(m.Url)
	r.Headers, r.Cookies = getHttpHeaders(m.ReqHeadersMap)
	// revisit below ?
	r.HeadersSize = -1
	r.BodySize = -1
	if len(m.RequestBody) > 0 {
		r.PostData = getPostData(m)
	}

	return &r
}

func addResponse(m *Message) *har.Response {
	r := har.Response{}
	r.Status = int64(m.Response)
	r.StatusText = http.StatusText(m.Response)
	r.HTTPVersion = http_version
	r.Headers, r.Cookies = getHttpHeaders(m.RespHeadersMap)
	location, ok := m.RespHeadersMap["Location"]
	if ok {
		r.RedirectURL = location[0]
	} else {
		location, ok = m.RespHeadersMap["location"]
		if ok {
			r.RedirectURL = location[0]
		}
	}

	r.Content = getResponseContent(m)
	// revisit ?
	r.HeadersSize = -1
	r.BodySize = -1
	return &r
}

func getQueryString(u string) []*har.NameValuePair {
	qValuePairs := []*har.NameValuePair{}
	uRL, err := url.Parse(u)
	if err != nil {
		log.Printf("ERROR: har.getQueryString error: %v\n", err)
		return qValuePairs
	}
	// Query keys can be included multiple times, just using the first occurence
	for k, v := range uRL.Query() {
		qValuePairs = append(qValuePairs, &har.NameValuePair{Name: k, Value: v[0]})
	}

	return qValuePairs
}

func getHttpHeaders(m map[string][]string) ([]*har.NameValuePair, []*har.Cookie) {
	nvp := []*har.NameValuePair{}
	cookies := []*har.Cookie{}
	// http request headers
	for k, v := range m {
		if strings.ToLower(k) == "cookie" {
			// http.request 'cookie' header
			for _, cookie := range strings.Split(v[0], ";") {
				cookieParts := strings.Split(cookie, "=")
				key := strings.TrimSpace(cookieParts[0])
				val := ""
				if len(cookieParts) > 1 {
					val = strings.TrimSpace(cookieParts[1])
				}
				cookies = append(cookies, &har.Cookie{Name: key, Value: val})
			}
		}
		if strings.ToLower(k) == "set-cookie" {
			// http.response 'Set-Cookie' header(s)
			for _, cookie := range v {
				// add each set-cookie to headers
				nvp = append(nvp, &har.NameValuePair{Name: k, Value: cookie})
				c := har.Cookie{}
				for _, attribute := range strings.Split(cookie, ";") {
					kv := strings.Split(attribute, "=")
					key := strings.ToLower(strings.TrimSpace(kv[0]))
					value := ""
					if len(kv) > 1 {
						value = strings.TrimSpace(kv[1])
					}

					switch {
					case key == "path":
						c.Path = value
					case key == "domain":
						c.Domain = value
					case key == "expires":
						c.Expires = value
					case key == "httponly":
						c.HTTPOnly = true
					case key == "secure":
						c.Secure = true
					case key == "comment":
						c.Comment = value
					default:
						c.Name = key
						c.Value = value
					}
				}
				cookies = append(cookies, &c)
			}
			continue
		}
		// add to headers
		nvp = append(nvp, &har.NameValuePair{Name: k, Value: v[0]})
	}

	return nvp, cookies
}

func getPostData(m *Message) *har.PostData {
	pd := har.PostData{}
	pd.Text = m.RequestBody
	mime, ok := m.ReqHeadersMap["Content-type"]
	if ok {
		pd.MimeType = mime[0]
	} else {
		mime, ok = m.ReqHeadersMap["content-type"]
		if ok {
			pd.MimeType = mime[0]
		} else {
			log.Printf("[WARN] ** .har Generation unknown content-type for POST\nHeaders: %+v\n", m.ReqHeadersMap)
			pd.MimeType = "unkown"
		}
	}

	if strings.ToLower(pd.MimeType) == "application/x-www-form-urlencoded" {
		params := []*har.Param{}
		for _, p := range strings.Split(pd.Text, "&") {
			kv := strings.Split(p, "=")
			if len(kv) > 0 {
				param := har.Param{Name: kv[0], Value: ""}
				if len(kv) > 1 {
					param.Value = kv[1]
				}
				params = append(params, &param)
			}
		}
		pd.Params = params
	}

	return &pd
}

func getResponseContent(m *Message) *har.Content {
	content := har.Content{}
	content.Size = int64(len(m.ResponseBody))
	mimeType, ok := m.RespHeadersMap["Content-Type"]
	if ok {
		content.MimeType = mimeType[0]
	} else {
		mimeType, ok = m.RespHeadersMap["content-type"]
		if ok {
			content.MimeType = mimeType[0]
		}
	}

	content.Text = m.ResponseBody
	return &content
}
