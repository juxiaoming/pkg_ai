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
 * 【科大讯飞】xfyun
 * Doc : https://www.xfyun.cn/doc/spark/HTTP%E8%B0%83%E7%94%A8%E6%96%87%E6%A1%A3.html#_1-%E6%8E%A5%E5%8F%A3%E8%AF%B4%E6%98%8E
 */

type XfYunConf struct {
	Url string `json:"url"`
	Key string `json:"key"`
}

func NewXfYunConf(url, key string) *Config {
	return &Config{XfYunUrl: url, XfYunKey: key}
}

type XfYunServer struct {
	Conf XfYunConf `json:"conf"`
}

func newXfYunServer(url, key string) *XfYunServer {
	return &XfYunServer{
		Conf: XfYunConf{
			Url: url,
			Key: key,
		},
	}
}

func (x *XfYunServer) Supplier() string {
	return "xfyun"
}

func (x *XfYunServer) RequestPath() string {
	return x.Conf.Url
}

type XfYunRequestBody struct {
	Messages    []Message `json:"messages"`
	Model       string    `json:"model"`
	MaxTokens   int64     `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
	Stream      bool      `json:"stream"`
}

func (x *XfYunServer) build(data RequestData, isStream bool) ([]byte, error) {
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

type XfYunChatResponse struct {
	Code    int64  `json:"code"`
	Message string `json:"message"`
	Sid     string `json:"sid"`
	Choices []struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		Index int64 `json:"index"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int64 `json:"prompt_tokens"`
		CompletionTokens int64 `json:"completion_tokens"`
		TotalTokens      int64 `json:"total_tokens"`
	} `json:"usage"`
	Error struct {
		Message string      `json:"message"`
		Type    string      `json:"type"`
		Param   interface{} `json:"param"`
		Code    string      `json:"code"`
	} `json:"error"`
}

func (x *XfYunServer) Chat(requestPath string, data []byte) (*Response, error) {
	headers := map[string]string{"Authorization": "Bearer " + x.Conf.Key}
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

	retStruct := XfYunChatResponse{}
	if err := json.Unmarshal(retBytes, &retStruct); err != nil {
		return ret, err
	}

	ret.PromptTokens = retStruct.Usage.PromptTokens
	ret.CompletionTokens = retStruct.Usage.CompletionTokens

	if retStruct.Message != "Success" {
		if len(retStruct.Message) > 0 {
			return ret, errors.New(retStruct.Message)
		} else {
			return ret, errors.New(retStruct.Error.Message)
		}
	}

	if len(retStruct.Choices) == 0 {
		return ret, errors.New("无有效响应数据")
	}

	ret.ResponseText = retStruct.Choices[0].Message.Content

	return ret, nil
}

type XfYunStreamResp struct {
	Code    int64  `json:"code"`
	Message string `json:"message"`
	Sid     string `json:"sid"`
	ID      string `json:"id"`
	Created int    `json:"created"`
	Choices []struct {
		Delta struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"delta"`
		Index int64 `json:"index"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int64 `json:"prompt_tokens"`
		CompletionTokens int64 `json:"completion_tokens"`
		TotalTokens      int64 `json:"total_tokens"`
	} `json:"usage"`
	Error struct {
		Message string      `json:"message"`
		Type    string      `json:"type"`
		Param   interface{} `json:"param"`
		Code    string      `json:"code"`
	} `json:"error"`
}

type XfYunErrorInfo struct {
	Message string `json:"message"`
	Error   struct {
		Message string      `json:"message"`
		Type    string      `json:"type"`
		Param   interface{} `json:"param"`
		Code    string      `json:"code"`
	} `json:"error"`
}

func (x *XfYunServer) ChatStream(requestPath string, data []byte, msgCh chan string, errChan chan error) (*Response, error) {
	headers := map[string]string{"Authorization": "Bearer " + x.Conf.Key}
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
				errStruct := &XfYunErrorInfo{}
				if err := json.Unmarshal(line, errStruct); err == nil {
					if len(errStruct.Message) > 0 {
						resErr = errors.New(errStruct.Message)
					} else {
						resErr = errors.New(errStruct.Error.Message)
					}
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

		retStruct := XfYunStreamResp{}
		if err := json.Unmarshal(line, &retStruct); err != nil {
			errChan <- err
			return ret, err
		}

		if retStruct.Message != "Success" {
			if len(retStruct.Message) > 0 {
				err = errors.New(retStruct.Message)
			} else {
				err = errors.New(retStruct.Error.Message)
			}
			errChan <- err
			return ret, err
		}

		if len(retStruct.Choices) == 0 {
			continue
		}
		ret.ResponseText += retStruct.Choices[0].Delta.Content
		msgCh <- retStruct.Choices[0].Delta.Content

		if retStruct.Usage.TotalTokens > 0 {
			ret.PromptTokens = retStruct.Usage.PromptTokens
			ret.CompletionTokens = retStruct.Usage.CompletionTokens
			close(msgCh)
			break
		}
	}

	return ret, nil
}
