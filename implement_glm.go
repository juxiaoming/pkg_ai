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
 * 【智谱清言】bigmodel
 * Doc : https://open.bigmodel.cn/dev/api/normal-model/glm-4
 */

type GlmConf struct {
	Url string `json:"url"`
	Key string `json:"key"`
}

func NewGlmConf(url, key string) *Config {
	return &Config{GlmUrl: url, GlmKey: key}
}

type GlmServer struct {
	Conf GlmConf `json:"conf"`
}

func newGlmServer(url, key string) *GlmServer {
	return &GlmServer{
		Conf: GlmConf{
			Url: url,
			Key: key,
		},
	}
}

func (g *GlmServer) Supplier() string {
	return "bigmodel"
}

func (g *GlmServer) RequestPath() string {
	return g.Conf.Url
}

type GlmRequestBody struct {
	Messages    []Message `json:"messages"`
	Model       string    `json:"model"`
	MaxTokens   int64     `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
	TopP        float64   `json:"top_p,omitempty"`
	Stream      bool      `json:"stream"`
}

func (g *GlmServer) build(data RequestData, isStream bool) ([]byte, error) {
	if data.UserQuery == "" || data.Model == "" {
		return []byte{}, errors.New("问题、模型为必传字段")
	}

	request := &GlmRequestBody{Stream: isStream, Messages: make([]Message, 0)}

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

	return json.Marshal(request)
}

type GlmChatResponse struct {
	Choices []struct {
		FinishReason string `json:"finish_reason"`
		Index        int64  `json:"index"`
		Message      struct {
			Content string `json:"content"`
			Role    string `json:"role"`
		} `json:"message"`
	} `json:"choices"`
	Created   int64  `json:"created"`
	Id        string `json:"id"`
	Model     string `json:"model"`
	RequestId string `json:"request_id"`
	Usage     struct {
		CompletionTokens int64 `json:"completion_tokens"`
		PromptTokens     int64 `json:"prompt_tokens"`
		TotalTokens      int64 `json:"total_tokens"`
	} `json:"usage"`
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func (g *GlmServer) Chat(requestPath string, data []byte) (*Response, error) {
	headers := map[string]string{"Authorization": "Bearer " + g.Conf.Key, "Content-Type": "application/json"}
	ret := &Response{RequestHeader: headers, RequestBody: data, ResponseData: make([]byte, 0)}

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

	retStruct := GlmChatResponse{}
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

type GlmStreamResp struct {
	Id      string `json:"id"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int64  `json:"index"`
		FinishReason string `json:"finish_reason"`
		Delta        struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int64 `json:"prompt_tokens"`
		CompletionTokens int64 `json:"completion_tokens"`
		TotalTokens      int64 `json:"total_tokens"`
	} `json:"usage"`
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type GlmErrorInfo struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func (g *GlmServer) ChatStream(requestPath string, data []byte, msgCh chan string, errChan chan error) (*Response, error) {
	headers := map[string]string{"Authorization": "Bearer " + g.Conf.Key, "Content-Type": "application/json"}
	ret := &Response{RequestHeader: headers, RequestBody: data, ResponseData: make([]byte, 0)}

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
				errStruct := &GlmErrorInfo{}
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

		retStruct := GlmStreamResp{}
		if err := json.Unmarshal(line, &retStruct); err != nil {
			errChan <- err
			return ret, err
		}
		if len(retStruct.Error.Message) > 0 {
			err := errors.New(retStruct.Error.Message)
			errChan <- err
			return ret, err
		}

		if len(retStruct.Choices) == 0 {
			continue
		}

		ret.ResponseText += retStruct.Choices[0].Delta.Content
		msgCh <- retStruct.Choices[0].Delta.Content

		if retStruct.Choices[0].FinishReason == "stop" {
			ret.PromptTokens = retStruct.Usage.PromptTokens
			ret.CompletionTokens = retStruct.Usage.CompletionTokens
			close(msgCh)
			break
		}
	}

	return ret, nil
}
