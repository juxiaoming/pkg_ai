package pkg_ai

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"github.com/golang-jwt/jwt/v4"
	"github.com/jinzhu/copier"
	"io"
	"time"
)

/**
 * 【商汤日日新】sensenova
 * Doc : https://platform.sensenova.cn/doc?path=/chat/ChatCompletions/ChatCompletions.md
 */

type SensenovaConf struct {
	Url          string `json:"url"`
	ClientId     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

func NewSensenovaConf(url, clientId, clientSecret string) *Config {
	return &Config{SensenovaUrl: url, SensenovaClientId: clientId, SensenovaClientSecret: clientSecret}
}

type SensenovaServer struct {
	Conf SensenovaConf `json:"conf"`
}

func newSensenovaServer(url, clientId, clientSecret string) *SensenovaServer {
	return &SensenovaServer{
		Conf: SensenovaConf{
			Url:          url,
			ClientId:     clientId,
			ClientSecret: clientSecret,
		},
	}
}

func (s *SensenovaServer) Supplier() string {
	return "sensenova"
}

func (s *SensenovaServer) RequestPath() string {
	return s.Conf.Url
}

type SensenovaRequestBody struct {
	Messages    []Message `json:"messages"`
	Model       string    `json:"model"`
	Temperature float64   `json:"temperature,omitempty"`
	TopP        float64   `json:"top_p,omitempty"`
	Stream      bool      `json:"stream"`
}

func (s *SensenovaServer) build(data RequestData, isStream bool) ([]byte, error) {
	if data.UserQuery == "" || data.Model == "" {
		return []byte{}, errors.New("问题、模型为必传字段")
	}

	request := &SensenovaRequestBody{Stream: isStream, Messages: make([]Message, 0)}

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

func (s *SensenovaServer) token(ak string, sk string) (string, error) {
	payload := jwt.MapClaims{
		"iss": ak,
		"exp": time.Now().Add(1800 * time.Second).Unix(),
		"nbf": time.Now().Add(-5 * time.Second).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)
	signedToken, err := token.SignedString([]byte(sk))
	if err != nil {
		return "", err
	}
	return signedToken, nil
}

type SensenovaChatResponse struct {
	Data struct {
		Id    string `json:"id"`
		Usage struct {
			PromptTokens     int64 `json:"prompt_tokens"`
			CompletionTokens int64 `json:"completion_tokens"`
			KnowledgeTokens  int64 `json:"knowledge_tokens"`
			TotalTokens      int64 `json:"total_tokens"`
		} `json:"usage"`
		Choices []struct {
			Index        int64  `json:"index"`
			Role         string `json:"role"`
			Message      string `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Plugins struct {
		} `json:"plugins"`
	} `json:"data"`
	Error struct {
		Message string `json:"message"`
		Code    int64  `json:"code"`
	} `json:"error"`
}

func (s *SensenovaServer) Chat(requestPath string, data []byte) (*Response, error) {
	ret := &Response{RequestData: data}

	token, err := s.token(s.Conf.ClientId, s.Conf.ClientSecret)
	if err != nil {
		return ret, err
	}

	headers := map[string]string{"Authorization": "Bearer " + token, "Content-Type": "application/json"}
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

	retStruct := SensenovaChatResponse{}
	if err := json.Unmarshal(retBytes, &retStruct); err != nil {
		return ret, err
	}

	ret.PromptTokens = retStruct.Data.Usage.PromptTokens
	ret.CompletionTokens = retStruct.Data.Usage.CompletionTokens

	if len(retStruct.Error.Message) > 0 {
		return ret, errors.New(retStruct.Error.Message)
	}

	if len(retStruct.Data.Choices) == 0 {
		return ret, errors.New("无有效响应数据")
	}

	ret.ResponseText = retStruct.Data.Choices[0].Message

	return ret, nil
}

type SensenovaStreamResp struct {
	Data struct {
		Id    string `json:"id"`
		Usage struct {
			PromptTokens     int64 `json:"prompt_tokens"`
			CompletionTokens int64 `json:"completion_tokens"`
			KnowledgeTokens  int64 `json:"knowledge_tokens"`
			TotalTokens      int64 `json:"total_tokens"`
		} `json:"usage"`
		Choices []struct {
			Index        int64  `json:"index"`
			Role         string `json:"role"`
			Delta        string `json:"delta"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Plugins struct {
		} `json:"plugins"`
	} `json:"data"`
	Status struct {
		Code    int64  `json:"code"`
		Message string `json:"message"`
	} `json:"status"`
	Error struct {
		Message string `json:"message"`
		Code    int64  `json:"code"`
	} `json:"error"`
}

type SensenovaErrorInfo struct {
	Error struct {
		Message string `json:"message"`
		Code    int64  `json:"code"`
	} `json:"error"`
}

func (s *SensenovaServer) ChatStream(requestPath string, data []byte, msgCh chan string, errChan chan error) (*Response, error) {
	ret := &Response{RequestData: data, ResponseData: make([]byte, 0)}

	token, err := s.token(s.Conf.ClientId, s.Conf.ClientSecret)
	if err != nil {
		return ret, err
	}

	headers := map[string]string{"Authorization": "Bearer " + token, "Content-Type": "application/json"}
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
				errStruct := &SensenovaErrorInfo{}
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

		retStruct := SensenovaStreamResp{}
		if err := json.Unmarshal(line, &retStruct); err != nil {
			errChan <- err
			return ret, err
		}

		if len(retStruct.Error.Message) > 0 {
			err = errors.New(retStruct.Error.Message)
			errChan <- err
			return ret, err
		}
		if retStruct.Status.Code != 0 {
			err = errors.New(retStruct.Status.Message)
			errChan <- err
			return ret, err
		}

		if len(retStruct.Data.Choices) == 0 {
			continue
		}

		ret.ResponseText += retStruct.Data.Choices[0].Delta
		msgCh <- retStruct.Data.Choices[0].Delta

		if retStruct.Data.Choices[0].FinishReason == "stop" {
			ret.PromptTokens = retStruct.Data.Usage.PromptTokens
			ret.CompletionTokens = retStruct.Data.Usage.CompletionTokens
			close(msgCh)
			break
		}
	}

	return ret, nil
}
