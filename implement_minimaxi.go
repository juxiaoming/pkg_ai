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
 * 【minimaxi】minimaxi
 * Doc : https://platform.minimaxi.com/document/ChatCompletion%20v2
 */

type MinimaxiConf struct {
	Url string `json:"url"`
	Key string `json:"key"`
}

func NewMinimaxiConf(url, key string) *Config {
	return &Config{MinimaxiUrl: url, MinimaxiKey: key}
}

type MinimaxiServer struct {
	Conf MinimaxiConf `json:"conf"`
}

func newMinimaxiServer(url, key string) *MinimaxiServer {
	return &MinimaxiServer{
		Conf: MinimaxiConf{
			Url: url,
			Key: key,
		},
	}
}

func (m *MinimaxiServer) Supplier() string {
	return "minimaxi"
}

func (m *MinimaxiServer) RequestPath() string {
	return m.Conf.Url
}

type MinimaxiRequestBody struct {
	Messages          []Message `json:"messages"`
	Model             string    `json:"model"`
	MaxTokens         int64     `json:"max_tokens,omitempty"`
	Temperature       float64   `json:"temperature,omitempty"`
	TopP              float64   `json:"top_p,omitempty"`
	N                 int64     `json:"n,omitempty"`
	MaskSensitiveInfo bool      `json:"mask_sensitive_info,omitempty"`
	Stream            bool      `json:"stream"`
}

func (m *MinimaxiServer) build(data RequestData, isStream bool) ([]byte, error) {
	if data.UserQuery == "" || data.Model == "" {
		return []byte{}, errors.New("问题、模型为必传字段")
	}

	request := &MinimaxiRequestBody{Stream: isStream, Messages: make([]Message, 0)}

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

type MinimaxiResponse struct {
	Id      string `json:"id"`
	Choices []struct {
		FinishReason string `json:"finish_reason"`
		Index        int    `json:"index"`
		Message      struct {
			Content      string `json:"content"`
			Role         string `json:"role"`
			Name         string `json:"name"`
			AudioContent string `json:"audio_content"`
		} `json:"message"`
	} `json:"choices"`
	Created int    `json:"created"`
	Model   string `json:"model"`
	Object  string `json:"object"`
	Usage   struct {
		TotalTokens     int64 `json:"total_tokens"`
		TotalCharacters int64 `json:"total_characters"`
	} `json:"usage"`
	InputSensitive      bool `json:"input_sensitive"`
	OutputSensitive     bool `json:"output_sensitive"`
	InputSensitiveType  int  `json:"input_sensitive_type"`
	OutputSensitiveType int  `json:"output_sensitive_type"`
	OutputSensitiveInt  int  `json:"output_sensitive_int"`
	BaseResp            struct {
		StatusCode int64  `json:"status_code"`
		StatusMsg  string `json:"status_msg"`
	} `json:"base_resp"`
}

func (m *MinimaxiServer) Chat(requestPath string, data []byte) (*Response, error) {
	ret := &Response{RequestData: data}
	headers := map[string]string{"Authorization": "Bearer " + m.Conf.Key, "Content-Type": "application/json"}
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

	retStruct := MinimaxiResponse{}
	if err := json.Unmarshal(retBytes, &retStruct); err != nil {
		return ret, err
	}

	ret.CompletionTokens = retStruct.Usage.TotalTokens

	if retStruct.BaseResp.StatusCode != 0 {
		return ret, errors.New(retStruct.BaseResp.StatusMsg)
	}

	if len(retStruct.Choices) == 0 {
		return ret, errors.New("无有效响应数据")
	}

	ret.ResponseText = retStruct.Choices[0].Message.Content

	return ret, nil
}

type MinimaxiErrorInfo struct {
	BaseResp struct {
		StatusCode int64  `json:"status_code"`
		StatusMsg  string `json:"status_msg"`
	} `json:"base_resp"`
}

type MinimaxiStreamResp struct {
	Id      string `json:"id"`
	Choices []struct {
		FinishReason string `json:"finish_reason"`
		Index        int    `json:"index"`
		Message      struct {
			Content      string `json:"content"`
			Role         string `json:"role"`
			Name         string `json:"name"`
			AudioContent string `json:"audio_content"`
		} `json:"message"`
	} `json:"choices"`
	Created int    `json:"created"`
	Model   string `json:"model"`
	Object  string `json:"object"`
	Usage   struct {
		TotalTokens     int64 `json:"total_tokens"`
		TotalCharacters int64 `json:"total_characters"`
	} `json:"usage"`
	InputSensitive      bool `json:"input_sensitive"`
	OutputSensitive     bool `json:"output_sensitive"`
	InputSensitiveType  int  `json:"input_sensitive_type"`
	OutputSensitiveType int  `json:"output_sensitive_type"`
	OutputSensitiveInt  int  `json:"output_sensitive_int"`
	BaseResp            struct {
		StatusCode int64  `json:"status_code"`
		StatusMsg  string `json:"status_msg"`
	} `json:"base_resp"`
}

func (m *MinimaxiServer) ChatStream(requestPath string, data []byte, msgCh chan string, errChan chan error, stopChan <-chan struct{}) (*Response, error) {
	ret := &Response{RequestData: data, ResponseData: make([]byte, 0)}

	headers := map[string]string{"Authorization": "Bearer " + m.Conf.Key, "Content-Type": "application/json"}
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
				errStruct := &MinimaxiErrorInfo{}
				if err := json.Unmarshal(line, errStruct); err == nil {
					resErr = errors.New(errStruct.BaseResp.StatusMsg)
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

		retStruct := MinimaxiStreamResp{}
		if err := json.Unmarshal(line, &retStruct); err != nil {
			errChan <- err
			return ret, err
		}

		if retStruct.BaseResp.StatusCode != 0 {
			errChan <- errors.New(retStruct.BaseResp.StatusMsg)
			return ret, errors.New(retStruct.BaseResp.StatusMsg)
		}

		if len(retStruct.Choices) == 0 {
			continue
		}
		ret.ResponseText += retStruct.Choices[0].Message.Content
		msgCh <- retStruct.Choices[0].Message.Content

		if retStruct.Usage.TotalTokens > 0 {
			ret.CompletionTokens = retStruct.Usage.TotalTokens
			close(msgCh)
			break
		}
	}

	return ret, nil
}
