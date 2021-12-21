package simba_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
)

var mockSlack *MockSlack

type Attachment struct {
	Fallback string `json:"fallback"`
	Color    string `json:"color"`
	Text     string `json:"text"`
}

type MockSlack struct {
	Server   *httptest.Server
	Received struct {
		Attachment []Attachment
		// ... define whatever you want to test against
	}
}

func New() *MockSlack {
	mockSlack = &MockSlack{Server: mockServer()}
	return mockSlack
}

func mockServer() *httptest.Server {
	handler := http.NewServeMux()
	handler.HandleFunc("/chat.postMessage", handlePostMessage)

	return httptest.NewServer(handler)
}

func parseAttachment(data string) []Attachment {
	a := make([]Attachment, 0)
	json.Unmarshal([]byte(data), &a)
	return a
}

func handlePostMessage(w http.ResponseWriter, r *http.Request) {
	body, _ := ioutil.ReadAll(r.Body)
	kvs := strings.Split(string(body), "&")
	// eg: channel=foo&mrkdwn=false&text=some+text&token=SCRET_TOKEN&unfurl_media=false

	m := make(map[string]string)

	for _, s := range kvs {
		kv := strings.Split(s, "=")
		s, err := url.QueryUnescape(kv[1])
		if err != nil {
			m[kv[0]] = kv[1]
		} else {
			m[kv[0]] = s
		}
	}

	mockSlack.Received.Attachment = parseAttachment(m["attachments"])

	// ref: https://api.slack.com/methods/chat.postMessage
	const response = `{
    "ok": true,
    "channel": "%s",
    "ts": "0000",
    "message": {
        "text": "%s",
        "username": "ecto1",
        "bot_id": "B19LU7CSY",
        "attachments": [
            {
                "text": "This is an attachment",
                "id": 1,
                "fallback": "This is an attachment's fallback"
            }
        ],
        "type": "message",
        "subtype": "bot_message",
        "ts": "1503435956.000247"
    }
 }`

	s := fmt.Sprintf(response, m["channel"], m["text"])
	_, _ = w.Write([]byte(s))
}
