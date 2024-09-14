package pkg_ai

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"time"
)

/**
 * 【混元大模型】hunyuan
 * Doc : https://cloud.tencent.com/document/api/1729/105701
 */

type HunyuanConf struct {
	Url          string `json:"url"`
	ClientId     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

func NewHunyuanConf(url, clientId, clientSecret string) *Config {
	return &Config{BaiDuUrl: url, HunyuanClientId: clientId, HunyuanClientSecret: clientSecret}
}

type HunyuanServer struct {
	Conf HunyuanConf `json:"conf"`
}

func newHunyuanServer(url, clientId, clientSecret string) *HunyuanServer {
	return &HunyuanServer{
		Conf: HunyuanConf{
			Url:          url,
			ClientId:     clientId,
			ClientSecret: clientSecret,
		},
	}
}

func (h *HunyuanServer) Supplier() string {
	return "hunyuan"
}

func (h *HunyuanServer) RequestPath() string {
	return h.Conf.Url
}

type HunyuanRequestBody struct {
	Model       string           `json:"Model"`
	Messages    []HunyuanMessage `json:"Messages"`
	Stream      bool             `json:"Stream"`
	TopP        float64          `json:"TopP,omitempty"`
	Temperature float64          `json:"Temperature,omitempty"`
}

func (h *HunyuanServer) build(data RequestData, isStream bool) ([]byte, error) {
	if data.UserQuery == "" || data.Model == "" {
		return []byte{}, errors.New("问题、模型为必传字段")
	}

	request := &HunyuanRequestBody{Stream: isStream, Messages: make([]HunyuanMessage, 0), Model: data.Model}

	if data.SystemQuery != "" {
		request.Messages = append(request.Messages, HunyuanMessage{Role: MessageSystem, Content: data.SystemQuery})
	}
	if data.History != nil && len(data.History) > 0 {
		for _, detail := range data.History {
			request.Messages = append(request.Messages, HunyuanMessage{Role: MessageUSer, Content: detail[0]})
			request.Messages = append(request.Messages, HunyuanMessage{Role: MessageAssistant, Content: detail[1]})
		}
	}
	request.Messages = append(request.Messages, HunyuanMessage{Role: MessageUSer, Content: data.UserQuery})

	if data.TopP > 0 {
		request.TopP = data.TopP
	}
	if data.Temperature > 0 {
		request.Temperature = data.Temperature
	}

	return json.Marshal(request)
}

func (h *HunyuanServer) token(payload []byte, timestamp int64) string {
	host := "hunyuan.tencentcloudapi.com"
	algorithm := "TC3-HMAC-SHA256"
	service := "hunyuan"

	// step 1: build canonical request string
	httpRequestMethod := "POST"
	canonicalURI := "/"
	canonicalQueryString := ""
	canonicalHeaders := fmt.Sprintf("content-type:%s\nhost:%s\n", "application/json", host)
	signedHeaders := "content-type;host"
	hashedRequestPayload := sha256hex(string(payload))
	canonicalRequest := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		httpRequestMethod,
		canonicalURI,
		canonicalQueryString,
		canonicalHeaders,
		signedHeaders,
		hashedRequestPayload)

	// step 2: build string to sign
	date := time.Unix(timestamp, 0).UTC().Format("2006-01-02")
	credentialScope := fmt.Sprintf("%s/%s/tc3_request", date, service)
	hashedCanonicalRequest := sha256hex(canonicalRequest)
	string2sign := fmt.Sprintf("%s\n%d\n%s\n%s",
		algorithm,
		timestamp,
		credentialScope,
		hashedCanonicalRequest)

	// step 3: sign string
	secretDate := hmacsha256(date, "TC3"+h.Conf.ClientSecret)
	secretService := hmacsha256(service, secretDate)
	secretSigning := hmacsha256("tc3_request", secretService)
	signature := hex.EncodeToString([]byte(hmacsha256(string2sign, secretSigning)))

	// step 4: build authorization
	authorization := fmt.Sprintf("%s Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		algorithm,
		h.Conf.ClientId,
		credentialScope,
		signedHeaders,
		signature)

	return authorization
}

type HunyuanChatResponse struct {
	Response struct {
		RequestID string `json:"RequestId"`
		Note      string `json:"Note"`
		Choices   []struct {
			Message struct {
				Role    string `json:"Role"`
				Content string `json:"Content"`
			} `json:"Message"`
			FinishReason string `json:"FinishReason"`
		} `json:"Choices"`
		Created int64  `json:"Created"`
		Id      string `json:"Id"`
		Usage   struct {
			PromptTokens     int64 `json:"PromptTokens"`
			CompletionTokens int64 `json:"CompletionTokens"`
			TotalTokens      int64 `json:"TotalTokens"`
		} `json:"Usage"`
		Error struct {
			Code    string `json:"Code"`
			Message string `json:"Message"`
		} `json:"Error"`
	} `json:"Response"`
}

func (h *HunyuanServer) Chat(requestPath string, data []byte) (*Response, error) {
	ret := &Response{RequestData: data}

	timestamp := time.Now().Unix()
	headers := map[string]string{
		"Authorization":  h.token(data, timestamp),
		"X-TC-Action":    "ChatCompletions",
		"X-TC-Version":   "2023-09-01",
		"X-TC-Timestamp": strconv.Itoa(int(timestamp)),
		"Host":           "hunyuan.tencentcloudapi.com",
		"content-type":   "application/json",
	}
	response, err := postBase(requestPath, string(data), headers)
	if err != nil {
		return ret, err
	}
	defer func() {
		_ = response.Body.Close()
	}()

	retBytes, err := io.ReadAll(response.Body)
	ret.ResponseData = retBytes
	if err != nil {
		return ret, err
	}

	retStruct := HunyuanChatResponse{}
	if err := json.Unmarshal(retBytes, &retStruct); err != nil {
		return ret, err
	}

	ret.PromptTokens = retStruct.Response.Usage.PromptTokens
	ret.CompletionTokens = retStruct.Response.Usage.CompletionTokens

	if len(retStruct.Response.Error.Message) > 0 {
		return ret, errors.New(retStruct.Response.Error.Message)
	}

	if len(retStruct.Response.Choices) == 0 {
		return ret, errors.New("无有效响应数据")
	}

	ret.ResponseText = retStruct.Response.Choices[0].Message.Content

	return ret, nil
}

type HunyuanErrorInfo struct {
	Response struct {
		Error struct {
			Code    string `json:"Code"`
			Message string `json:"Message"`
		} `json:"Error"`
		RequestID string `json:"RequestId"`
	} `json:"Response"`
}

type HunyuanStreamResp struct {
	Note    string `json:"Note"`
	Choices []struct {
		Delta struct {
			Role    string `json:"Role"`
			Content string `json:"Content"`
		} `json:"Delta"`
		FinishReason string `json:"FinishReason"`
	} `json:"Choices"`
	Created int64  `json:"Created"`
	Id      string `json:"Id"`
	Usage   struct {
		PromptTokens     int64 `json:"PromptTokens"`
		CompletionTokens int64 `json:"CompletionTokens"`
		TotalTokens      int64 `json:"TotalTokens"`
	} `json:"Usage"`
	Response struct {
		Error struct {
			Code    string `json:"Code"`
			Message string `json:"Message"`
		} `json:"Error"`
		RequestID string `json:"RequestId"`
	} `json:"Response"`
}

func (h *HunyuanServer) ChatStream(requestPath string, data []byte, msgCh chan string, errChan chan error) (*Response, error) {
	ret := &Response{RequestData: data, ResponseData: make([]byte, 0)}

	timestamp := time.Now().Unix()
	headers := map[string]string{
		"Authorization":  h.token(data, timestamp),
		"X-TC-Action":    "ChatCompletions",
		"X-TC-Version":   "2023-09-01",
		"X-TC-Timestamp": strconv.Itoa(int(timestamp)),
		"Host":           "hunyuan.tencentcloudapi.com",
		"content-type":   "application/json",
	}
	response, err := postBase(requestPath, string(data), headers)
	if err != nil {
		errChan <- err
		return ret, err
	}
	defer func() {
		_ = response.Body.Close()
	}()

	reader := bufio.NewReader(response.Body)

	for {
		line, err := reader.ReadBytes('\n')
		ret.ResponseData = append(ret.ResponseData, line...)
		line = bytes.TrimSuffix(line, []byte("\n"))

		if err != nil {

			if errors.Is(err, io.EOF) {
				resErr := err
				errStruct := &HunyuanErrorInfo{}
				if err := json.Unmarshal(line, errStruct); err == nil {
					resErr = errors.New(errStruct.Response.Error.Message)
				}

				errChan <- resErr
				return ret, resErr
			}

			errChan <- err
			return ret, err
		}

		if string(line) == "" {
			continue
		}

		headerData := []byte("data: ")
		if !bytes.HasPrefix(line, headerData) {
			continue
		}
		line = bytes.TrimPrefix(line, headerData)

		retStruct := HunyuanStreamResp{}
		if err := json.Unmarshal(line, &retStruct); err != nil {
			errChan <- err
			return ret, err
		}
		if len(retStruct.Response.Error.Message) > 0 {
			err := errors.New(retStruct.Response.Error.Message)
			errChan <- err
			return ret, err
		}

		if len(retStruct.Choices) == 0 {
			continue
		}

		if retStruct.Choices[0].FinishReason == "stop" {
			ret.PromptTokens = retStruct.Usage.PromptTokens
			ret.CompletionTokens = retStruct.Usage.CompletionTokens
			close(msgCh)
			break
		} else {
			ret.ResponseText += retStruct.Choices[0].Delta.Content
			msgCh <- retStruct.Choices[0].Delta.Content
		}
	}

	return ret, nil
}
