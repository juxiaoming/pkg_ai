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
 * 【百川智能】baichuan
 * Doc : https://platform.baichuan-ai.com/docs/api
 */

type BaiChuanConf struct {
	Url string `json:"url"`
	Key string `json:"key"`
}

func NewBaiChuanConf(url, key string) *Config {
	return &Config{BaiChuanUrl: url, BaiChuanKey: key}
}

type BaiChuanServer struct {
	Conf BaiChuanConf `json:"conf"`
}

func newBaiChuanServer(url, key string) *BaiChuanServer {
	return &BaiChuanServer{
		Conf: BaiChuanConf{
			Url: url,
			Key: key,
		},
	}
}

func (b *BaiChuanServer) Supplier() string {
	return "baichuan"
}

func (b *BaiChuanServer) RequestPath() string {
	return b.Conf.Url
}

type BaiChuanRequestBody struct {
	Messages    []Message `json:"messages"`
	Model       string    `json:"model"`
	MaxTokens   int64     `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
	TopP        float64   `json:"top_p,omitempty"`
	Stream      bool      `json:"stream"`
}

func (b *BaiChuanServer) build(data RequestData, isStream bool) ([]byte, error) {
	if data.UserQuery == "" || data.Model == "" {
		return []byte{}, errors.New("问题、模型为必传字段")
	}

	request := &XfYunRequestBody{Stream: isStream, Messages: make([]Message, 0)}

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

type BaiChuanChatResponse struct {
	Id      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int64 `json:"index"`
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
		SearchCount      int64 `json:"search_count"`
	} `json:"usage"`
	Error struct {
		Message string      `json:"message"`
		Type    string      `json:"type"`
		Param   interface{} `json:"param"`
		Code    string      `json:"code"`
	} `json:"error"`
}

func (b *BaiChuanServer) Chat(requestPath string, data []byte) (*Response, error) {
	headers := map[string]string{"Authorization": "Bearer " + b.Conf.Key, "Content-Type": "application/json"}
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

	retStruct := BaiChuanChatResponse{}
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

type BaiChuanStreamResp struct {
	Id      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index int64 `json:"index"`
		Delta struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int64 `json:"prompt_tokens"`
		CompletionTokens int64 `json:"completion_tokens"`
		TotalTokens      int64 `json:"total_tokens"`
		SearchCount      int64 `json:"search_count"`
	} `json:"usage"`
	Error struct {
		Code    string      `json:"code"`
		Param   interface{} `json:"param"`
		Type    string      `json:"type"`
		Message string      `json:"message"`
	} `json:"error"`
}

type BaiChuanErrorInfo struct {
	Error struct {
		Message string      `json:"message"`
		Type    string      `json:"type"`
		Param   interface{} `json:"param"`
		Code    string      `json:"code"`
	} `json:"error"`
}

func (b *BaiChuanServer) ChatStream(requestPath string, data []byte, msgCh chan string, errChan chan error) (*Response, error) {
	headers := map[string]string{"Authorization": "Bearer " + b.Conf.Key, "Content-Type": "application/json"}
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
				errStruct := &BaiChuanErrorInfo{}
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

		retStruct := BaiChuanStreamResp{}
		if err := json.Unmarshal(line, &retStruct); err != nil {
			errChan <- err
			return ret, err
		}

		if len(retStruct.Error.Message) > 0 {
			err = errors.New(retStruct.Error.Message)
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
