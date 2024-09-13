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
 * 【通义千问】qwen
 * Doc : https://help.aliyun.com/zh/dashscope/developer-reference/use-qwen
 */

type QwenConf struct {
	Url string `json:"url"`
	Key string `json:"key"`
}

func NewQwenConf(url, key string) *Config {
	return &Config{QwenUrl: url, QwenKey: key}
}

type QwenServer struct {
	Conf QwenConf `json:"conf"`
}

func newQwenServer(url, key string) *QwenServer {
	return &QwenServer{
		Conf: QwenConf{
			Url: url,
			Key: key,
		},
	}
}

func (q *QwenServer) Supplier() string {
	return "qwen"
}

func (q *QwenServer) RequestPath() string {
	return q.Conf.Url
}

type QwenRequestBody struct {
	Messages      []Message     `json:"messages"`
	Model         string        `json:"model"`
	MaxTokens     int64         `json:"max_tokens,omitempty"`
	Temperature   float64       `json:"temperature,omitempty"`
	TopP          float64       `json:"top_p,omitempty"`
	StreamOptions StreamOptions `json:"stream_options,omitempty"`
	Stream        bool          `json:"stream"`
}

func (q *QwenServer) build(data RequestData, isStream bool) ([]byte, error) {
	if data.UserQuery == "" || data.Model == "" {
		return []byte{}, errors.New("问题、模型为必传字段")
	}

	request := &QwenRequestBody{Stream: isStream, Messages: make([]Message, 0)}

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

type QwenChatResponse struct {
	Choices []struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string      `json:"finish_reason"`
		Index        int64       `json:"index"`
		Logprobs     interface{} `json:"logprobs"`
	} `json:"choices"`
	Object string `json:"object"`
	Usage  struct {
		PromptTokens     int64 `json:"prompt_tokens"`
		CompletionTokens int64 `json:"completion_tokens"`
		TotalTokens      int64 `json:"total_tokens"`
	} `json:"usage"`
	Created           int64       `json:"created"`
	SystemFingerprint interface{} `json:"system_fingerprint"`
	Model             string      `json:"model"`
	Id                string      `json:"id"`
	Error             struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

func (q *QwenServer) Chat(requestPath string, data []byte) (*Response, error) {
	ret := &Response{RequestData: data}
	headers := map[string]string{"Authorization": "Bearer " + q.Conf.Key}
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

	retStruct := QwenChatResponse{}
	if err := json.Unmarshal(retBytes, &retStruct); err != nil {
		return ret, err
	}

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

type QwenResponse struct {
	Choices []struct {
		FinishReason string `json:"finish_reason"`
		Delta        struct {
			Content string `json:"content"`
		} `json:"delta"`
		Index    int64       `json:"index"`
		Logprobs interface{} `json:"logprobs"`
	} `json:"choices"`
	Object string `json:"object"`
	Usage  struct {
		PromptTokens     int64 `json:"prompt_tokens"`
		CompletionTokens int64 `json:"completion_tokens"`
		TotalTokens      int64 `json:"total_tokens"`
	} `json:"usage"`
	Created           int64       `json:"created"`
	SystemFingerprint interface{} `json:"system_fingerprint"`
	Model             string      `json:"model"`
	Id                string      `json:"id"`
	Error             struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

type QwenErrorInfo struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

func (q *QwenServer) ChatStream(requestPath string, data []byte, msgCh chan string, errChan chan error, stopChan <-chan struct{}) (*Response, error) {
	ret := &Response{RequestData: data, ResponseData: make([]byte, 0)}

	headers := map[string]string{"Authorization": "Bearer " + q.Conf.Key, "Content-Type": "application/json"}
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
				errStruct := &QwenErrorInfo{}
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

		retStruct := QwenResponse{}
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
