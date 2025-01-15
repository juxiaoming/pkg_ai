package pkg_ai

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
)

type DeepSeekConf struct {
	Url string `json:"url"`
	Key string `json:"key"`
}

func NewDeepSeekConf(url, key string) *Config {
	return &Config{DeepSeekUrl: url, DeepSeekKey: key}
}

type DeepSeekServer struct {
	Conf DeepSeekConf `json:"conf"`
}

func newDeepSeekServer(url, key string) *DeepSeekServer {
	return &DeepSeekServer{
		Conf: DeepSeekConf{
			Url: url,
			Key: key,
		},
	}
}

func (d *DeepSeekServer) Supplier() string {
	return "deepSeek"
}

func (d *DeepSeekServer) RequestPath() string {
	return d.Conf.Url
}

type DeepSeekRequestBody struct {
	Messages    []Message `json:"messages"`
	Model       string    `json:"model"`
	MaxTokens   int64     `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
	Stream      bool      `json:"stream"`
}

func (d *DeepSeekServer) build(data RequestData, isStream bool) ([]byte, error) {
	if data.UserQuery == "" || data.Model == "" {
		return []byte{}, errors.New("问题、模型为必传字段")
	}

	request := &DeepSeekRequestBody{Stream: isStream, Messages: make([]Message, 0), Model: data.Model}

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

type DeepSeekChatResponse struct {
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

func (d *DeepSeekServer) Chat(requestPath string, data []byte) (*Response, error) {
	headers := map[string]string{"Authorization": "Bearer " + d.Conf.Key, "content-type": "application/json"}
	ret := &Response{RequestHeader: make([]byte, 0), RequestBody: data, ResponseData: make([][]byte, 0)}
	ret.RequestHeader, _ = json.Marshal(headers)

	response, err := postBase(requestPath, string(data), headers)
	if err != nil {
		return ret, err
	}
	defer func() {
		_ = response.Body.Close()
	}()

	retBytes, err := io.ReadAll(response.Body)
	ret.ResponseData = append(ret.ResponseData, retBytes)
	if err != nil {
		return ret, err
	}

	retStruct := DeepSeekChatResponse{}
	if err := json.Unmarshal(retBytes, &retStruct); err != nil {
		return ret, err
	}

	ret.RequestId = retStruct.Id
	ret.PromptTokens = retStruct.Usage.PromptTokens
	ret.CompletionTokens = retStruct.Usage.CompletionTokens

	if len(retStruct.Error.Message) > 0 {
		return ret, errors.New(retStruct.Error.Message)
	}

	if len(retStruct.Choices) == 0 {
		return ret, errors.New("无有效响应数据")
	}

	ret.ResponseText = retStruct.Choices[0].Message.Content

	return ret, nil
}

type DeepSeekErrorInfo struct {
	Error struct {
		Code    int64  `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type DeepSeekStreamResp struct {
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
	} `json:"choices"`
	Usage struct {
		PromptTokens     int64 `json:"prompt_tokens"`
		CompletionTokens int64 `json:"completion_tokens"`
		TotalTokens      int64 `json:"total_tokens"`
	} `json:"usage"`
}

func (d *DeepSeekServer) ChatStream(requestPath string, data []byte, msgCh chan string, errChan chan error) (*Response, error) {
	headers := map[string]string{"Authorization": "Bearer " + d.Conf.Key, "content-type": "application/json"}
	ret := &Response{RequestHeader: make([]byte, 0), RequestBody: data, ResponseData: make([][]byte, 0)}
	ret.RequestHeader, _ = json.Marshal(headers)

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
		ret.ResponseData = append(ret.ResponseData, line)
		line = bytes.TrimSuffix(line, []byte("\n"))

		if err != nil {
			if errors.Is(err, io.EOF) {
				resErr := err
				errStruct := &DeepSeekErrorInfo{}
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

		retStruct := DeepSeekStreamResp{}
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
			ret.RequestId = retStruct.Id
			ret.PromptTokens = retStruct.Usage.PromptTokens
			ret.CompletionTokens = retStruct.Usage.CompletionTokens
			close(msgCh)
			break
		}
	}

	return ret, nil
}
