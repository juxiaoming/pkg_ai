package pkg_ai

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
)

func postBase(url string, payload string, headers map[string]string) (resp *http.Response, err error) {
	client := &http.Client{}
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
	client := &http.Client{}
	req, err := http.NewRequest("GET", requestUrl, nil)
	if err != nil {
		return
	}
	for index, val := range headers {
		req.Header.Set(index, val)
	}
	return client.Do(req)
}

func sha256hex(s string) string {
	b := sha256.Sum256([]byte(s))
	return hex.EncodeToString(b[:])
}

func hmacsha256(s, key string) string {
	hashed := hmac.New(sha256.New, []byte(key))
	hashed.Write([]byte(s))
	return string(hashed.Sum(nil))
}
