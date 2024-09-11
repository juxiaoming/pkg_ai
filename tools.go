package pkg_ai

import (
	"net/http"
	"strings"
	"time"
)

func postBase(url string, payload string, headers map[string]string) (resp *http.Response, err error) {
	client := &http.Client{Timeout: time.Second * 5}
	req, err := http.NewRequest("POST", url, strings.NewReader(payload))
	if err != nil {
		return
	}
	for index, val := range headers {
		req.Header.Set(index, val)
	}

	return client.Do(req)
}

func getBase(requestUrl string, headers map[string]string) (resp *http.Response, err error) {
	client := &http.Client{Timeout: time.Second * 5}
	req, err := http.NewRequest("GET", requestUrl, nil)
	if err != nil {
		return
	}
	for index, val := range headers {
		req.Header.Set(index, val)
	}
	return client.Do(req)
}
