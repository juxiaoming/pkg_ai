package pkg_ai

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"github.com/jinzhu/copier"
	"io"
)

/**
 * 【月之暗面】moonshot
 * Doc : https://platform.moonshot.cn/docs/api/chat#%E5%85%AC%E5%BC%80%E7%9A%84%E6%9C%8D%E5%8A%A1%E5%9C%B0%E5%9D%80
 */

type MoonshotConf struct {
	Url string `json:"url"`
	Key string `json:"key"`
}

func NewMoonshotConf(url, key string) *Config {
	return &Config{MoonshotUrl: url, MoonshotKey: key}
}

type MoonshotServer struct {
	Conf MoonshotConf `json:"conf"`
}

func newMoonshotServer(url, key string) *MoonshotServer {
	return &MoonshotServer{
		Conf: MoonshotConf{
			Url: url,
			Key: key,
		},
	}
}

func (m *MoonshotServer) Supplier() string {
	return "moonshot"
}

func (m *MoonshotServer) RequestPath() string {
	return m.Conf.Url
}

type MoonshotRequestBody struct {
	Messages         []Message `json:"messages"`
	Model            string    `json:"model"`
	MaxTokens        int64     `json:"max_tokens,omitempty"`
	Temperature      float64   `json:"temperature,omitempty"`
	TopP             float64   `json:"top_p,omitempty"`
	N                int64     `json:"n,omitempty"`
	PresencePenalty  float64   `json:"presence_penalty,omitempty"`
	FrequencyPenalty float64   `json:"frequency_penalty,omitempty"`
	ResponseFormat   struct {
		Type string `json:"type"`
	} `json:"response_format,omitempty"`
	Stop   []string `json:"stop,omitempty"`
	Stream bool     `json:"stream"`
}

func (m *MoonshotServer) build(data RequestData, isStream bool) ([]byte, error) {
	if data.UserQuery == "" || data.Model == "" {
		return []byte{}, errors.New("问题、模型为必传字段")
	}

	request := &MoonshotRequestBody{Stream: isStream, Messages: make([]Message, 0), Stop: make([]string, 0), ResponseFormat: struct {
		Type string `json:"type"`
	}(struct{ Type string }{Type: "text"})}

	if err := copier.Copy(request, &data); err != nil {
		return nil, err
	}

	if data.ResponseFormat == "json" {
		request.ResponseFormat = struct {
			Type string `json:"type"`
		}(struct{ Type string }{Type: "json_object"})
	}

	if data.SystemQuery != "" {
		request.Messages = append(request.Messages, Message{Role: MessageSystem, Content: data.SystemQuery})
	}
	if data.History != nil && len(data.History) > 0 {
		for _, detail := range data.History {
			request.Messages = append(request.Messages, Message{Role: MessageUSer, Content: detail[0]})
			request.Messages = append(request.Messages, Message{Role: MessageAssistant, Content: detail[1]})
		}
	}
	request.Messages = append(request.Messages, Message{Role: MessageUSer, Content: data.UserQuery})

	return json.Marshal(request)
}

type MoonshotChatResponse struct {
	Id      string `json:"id"`
	Object  string `json:"object"`
	Created int    `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int64 `json:"prompt_tokens"`
		CompletionTokens int64 `json:"completion_tokens"`
		TotalTokens      int64 `json:"total_tokens"`
	} `json:"usage"`
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

func (m *MoonshotServer) Chat(requestPath string, data []byte) (*Response, error) {
	ret := &Response{RequestData: data}
	headers := map[string]string{"Authorization": "Bearer " + m.Conf.Key}
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

	retStruct := MoonshotChatResponse{}
	if err := json.Unmarshal(retBytes, &retStruct); err != nil {
		return ret, err
	}

	ret.PromptTokens = retStruct.Usage.PromptTokens
	ret.CompletionTokens = retStruct.Usage.CompletionTokens

	if retStruct.Error.Message != "" {
		return ret, errors.New(retStruct.Error.Message)
	}

	if len(retStruct.Choices) == 0 {
		return ret, errors.New("无有效响应数据")
	}

	ret.ResponseText = retStruct.Choices[0].Message.Content

	return ret, nil
}

type MoonshotErrorInfo struct {
	Error struct {
		Code    int64  `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type MoonshotStreamResp struct {
	Id      string `json:"id"`
	Object  string `json:"object"`
	Created int    `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index int `json:"index"`
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
		Usage        struct {
			PromptTokens     int64 `json:"prompt_tokens"`
			CompletionTokens int64 `json:"completion_tokens"`
			TotalTokens      int64 `json:"total_tokens"`
		} `json:"usage"`
	} `json:"choices"`
	SystemFingerprint string `json:"system_fingerprint"`
}

func (m *MoonshotServer) ChatStream(requestPath string, data []byte, msgCh chan string, errChan chan error, stopChan <-chan struct{}) (*Response, error) {
	ret := &Response{RequestData: data, ResponseData: make([]byte, 0)}

	headers := map[string]string{"Authorization": "Bearer " + m.Conf.Key}
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
				errStruct := &MoonshotErrorInfo{}
				if err := json.Unmarshal(line, errStruct); err == nil {
					resErr = errors.New(errStruct.Error.Message)
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

		if string(line) == "[DONE]" {
			close(msgCh)
			break
		}

		retStruct := MoonshotStreamResp{}
		if err := json.Unmarshal(line, &retStruct); err != nil {
			errChan <- err
			return ret, err
		}
		if len(retStruct.Choices) == 0 {
			continue
		}
		ret.ResponseText += retStruct.Choices[0].Delta.Content
		msgCh <- retStruct.Choices[0].Delta.Content

		if retStruct.Choices[0].FinishReason == "stop" {
			ret.PromptTokens = retStruct.Choices[0].Usage.PromptTokens
			ret.CompletionTokens = retStruct.Choices[0].Usage.CompletionTokens
		}
	}

	return ret, nil
}
