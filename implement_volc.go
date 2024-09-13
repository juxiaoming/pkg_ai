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
 * 【抖音豆包】volcengine
 * Doc : https://www.volcengine.com/docs/82379/1298454
 */

type VolcConf struct {
	Url string `json:"url"`
	Key string `json:"key"`
}

func NewVolcConf(url, key string) *Config {
	return &Config{VolcUrl: url, VolcKey: key}
}

type VolcServer struct {
	Conf VolcConf `json:"conf"`
}

func newVolcServer(url, key string) *VolcServer {
	return &VolcServer{
		Conf: VolcConf{
			Url: url,
			Key: key,
		},
	}
}

func (m *VolcServer) Supplier() string {
	return "volc"
}

func (m *VolcServer) RequestPath() string {
	return m.Conf.Url
}

type VolcRequestBody struct {
	Messages      []Message     `json:"messages"`
	Model         string        `json:"model"`
	MaxTokens     int64         `json:"max_tokens,omitempty"`
	Temperature   float64       `json:"temperature,omitempty"`
	TopP          float64       `json:"top_p,omitempty"`
	Stop          []string      `json:"stop,omitempty"`
	Stream        bool          `json:"stream"`
	StreamOptions StreamOptions `json:"stream_options,omitempty"`
}

type StreamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

func (m *VolcServer) build(data RequestData, isStream bool) ([]byte, error) {
	if data.UserQuery == "" || data.Model == "" {
		return []byte{}, errors.New("问题、模型为必传字段")
	}

	request := &VolcRequestBody{Stream: isStream, Messages: make([]Message, 0), Stop: make([]string, 0)}

	if err := copier.Copy(request, &data); err != nil {
		return nil, err
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
	if isStream {
		request.StreamOptions = StreamOptions{IncludeUsage: true}
	}

	return json.Marshal(request)
}

type VolcChatResponse struct {
	Choices []struct {
		FinishReason string `json:"finish_reason"`
		Index        int64  `json:"index"`
		Message      struct {
			Content string `json:"content"`
			Role    string `json:"role"`
		} `json:"message"`
	} `json:"choices"`
	Created int    `json:"created"`
	Id      string `json:"id"`
	Model   string `json:"model"`
	Object  string `json:"object"`
	Usage   struct {
		CompletionTokens int64 `json:"completion_tokens"`
		PromptTokens     int64 `json:"prompt_tokens"`
		TotalTokens      int64 `json:"total_tokens"`
	} `json:"usage"`
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Param   string `json:"param"`
		Type    string `json:"type"`
	} `json:"error"`
}

func (m *VolcServer) Chat(requestPath string, data []byte) (*Response, error) {
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

	retStruct := VolcChatResponse{}
	if err := json.Unmarshal(retBytes, &retStruct); err != nil {
		return ret, err
	}

	ret.PromptTokens = retStruct.Usage.PromptTokens
	ret.CompletionTokens = retStruct.Usage.CompletionTokens

	if len(retStruct.Error.Code) > 0 {
		return ret, errors.New(retStruct.Error.Message)
	}

	if len(retStruct.Choices) == 0 {
		return ret, errors.New("无有效响应数据")
	}

	ret.ResponseText = retStruct.Choices[0].Message.Content

	return ret, nil
}

type VolcStreamResp struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
			Role    string `json:"role"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
		Index        int    `json:"index"`
	} `json:"choices"`
	Created int64  `json:"created"`
	Id      string `json:"id"`
	Model   string `json:"model"`
	Object  string `json:"object"`
	Usage   struct {
		CompletionTokens int64 `json:"completion_tokens"`
		PromptTokens     int64 `json:"prompt_tokens"`
		TotalTokens      int64 `json:"total_tokens"`
	} `json:"usage"`
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Param   string `json:"param"`
		Type    string `json:"type"`
	} `json:"error"`
}

type VolcErrorInfo struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Param   string `json:"param"`
		Type    string `json:"type"`
	} `json:"error"`
}

func (m *VolcServer) ChatStream(requestPath string, data []byte, msgCh chan string, errChan chan error, stopChan <-chan struct{}) (*Response, error) {
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
				errStruct := &VolcErrorInfo{}
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

		retStruct := VolcStreamResp{}
		if err := json.Unmarshal(line, &retStruct); err != nil {
			errChan <- err
			return ret, err
		}
		if len(retStruct.Error.Message) > 0 {
			err := errors.New(retStruct.Error.Message)
			errChan <- err
			return ret, err
		}

		if retStruct.Usage.TotalTokens > 0 {
			ret.PromptTokens = retStruct.Usage.PromptTokens
			ret.CompletionTokens = retStruct.Usage.CompletionTokens
		}

		if len(retStruct.Choices) == 0 {
			continue
		}
		ret.ResponseText += retStruct.Choices[0].Delta.Content
		msgCh <- retStruct.Choices[0].Delta.Content
	}

	return ret, nil
}
