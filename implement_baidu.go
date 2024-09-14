package pkg_ai

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"github.com/jinzhu/copier"
	"io"
	"net/url"
	"sync"
	"time"
)

/**
 * 【文心一言】baidubce
 * Doc : https://cloud.baidu.com/doc/WENXINWORKSHOP/s/clntwmv7t#http%E8%B0%83%E7%94%A8
 */

const BaiDuTokenUrl = "https://aip.baidubce.com/oauth/2.0/token"

var (
	BaiDuToken    string = ""
	BaiDuTokenExp int64  = 0
	BaiDuLock     sync.Mutex
)

type BaiDuConf struct {
	Url          string `json:"url"`
	ClientId     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

func NewBaiDuConf(url, clientId, clientSecret string) *Config {
	return &Config{BaiDuUrl: url, BaiDuClientId: clientId, BaiDuClientSecret: clientSecret}
}

type BaiDuServer struct {
	Conf BaiDuConf `json:"conf"`
}

func newBaiDuServer(url, clientId, clientSecret string) *BaiDuServer {
	return &BaiDuServer{
		Conf: BaiDuConf{
			Url:          url,
			ClientId:     clientId,
			ClientSecret: clientSecret,
		},
	}
}

func (b *BaiDuServer) Supplier() string {
	return "baidubce"
}

func (b *BaiDuServer) RequestPath() string {
	return b.Conf.Url
}

type BaiDuRequestBody struct {
	Messages    []Message `json:"messages"`
	Model       string    `json:"model"`
	Temperature float64   `json:"temperature,omitempty"`
	TopP        float64   `json:"top_p,omitempty"`
	Stream      bool      `json:"stream"`
}

func (b *BaiDuServer) build(data RequestData, isStream bool) ([]byte, error) {
	if data.UserQuery == "" || data.Model == "" {
		return []byte{}, errors.New("问题、模型为必传字段")
	}

	request := &BaiDuRequestBody{Stream: isStream, Messages: make([]Message, 0)}

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

type BaiDuTokenResponse struct {
	RefreshToken     string `json:"refresh_token"`
	ExpiresIn        int    `json:"expires_in"`
	SessionKey       string `json:"session_key"`
	AccessToken      string `json:"access_token"`
	Scope            string `json:"scope"`
	SessionSecret    string `json:"session_secret"`
	ErrorDescription string `json:"error_description"`
	Error            string `json:"error"`
}

func (b *BaiDuServer) token() (string, error) {
	formData := url.Values{}
	formData.Set("grant_type", "client_credentials")
	formData.Set("client_id", b.Conf.ClientId)
	formData.Set("client_secret", b.Conf.ClientSecret)

	headers := map[string]string{"Accept": "application/json", "Content-Type": "application/x-www-form-urlencoded"}
	response, err := postBase(BaiDuTokenUrl, formData.Encode(), headers)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = response.Body.Close()
	}()

	responseStruct := &BaiDuTokenResponse{}
	if err := json.NewDecoder(response.Body).Decode(responseStruct); err != nil {
		return "", err
	}

	if len(responseStruct.Error) != 0 {
		return "", errors.New(responseStruct.ErrorDescription)
	}

	return responseStruct.AccessToken, nil
}

func (b *BaiDuServer) Token() (string, error) {
	BaiDuLock.Lock()
	defer BaiDuLock.Unlock()

	if time.Now().Unix() < BaiDuTokenExp {
		return BaiDuToken, nil
	}

	token, err := b.token()
	if err != nil {
		return token, err
	}
	BaiDuTokenExp = time.Now().Unix() + 86400*30 - 7200
	BaiDuToken = token

	return token, nil
}

type BaiDuResponse struct {
	Id               string `json:"id"`
	Object           string `json:"object"`
	Created          int    `json:"created"`
	Result           string `json:"result"`
	IsTruncated      bool   `json:"is_truncated"`
	NeedClearHistory bool   `json:"need_clear_history"`
	FinishReason     string `json:"finish_reason"`
	Usage            struct {
		PromptTokens     int64 `json:"prompt_tokens"`
		CompletionTokens int64 `json:"completion_tokens"`
		TotalTokens      int64 `json:"total_tokens"`
	} `json:"usage"`
	ErrorCode int64  `json:"error_code"`
	ErrorMsg  string `json:"error_msg"`
}

func (b *BaiDuServer) Chat(requestPath string, data []byte) (*Response, error) {
	ret := &Response{RequestData: data}

	token, err := b.Token()
	if err != nil {
		return ret, err
	}
	requestPath = requestPath + "?access_token=" + token

	headers := map[string]string{"Content-Type": "application/json"}
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

	retStruct := BaiDuResponse{}
	if err := json.Unmarshal(retBytes, &retStruct); err != nil {
		return ret, err
	}

	if retStruct.ErrorCode != 0 {
		return ret, errors.New(retStruct.ErrorMsg)
	}

	ret.PromptTokens = retStruct.Usage.PromptTokens
	ret.CompletionTokens = retStruct.Usage.CompletionTokens

	ret.ResponseText = retStruct.Result

	return ret, nil
}

type BaiDuStreamResp struct {
	Id               string `json:"id"`
	Object           string `json:"object"`
	Created          int64  `json:"created"`
	SentenceID       int64  `json:"sentence_id"`
	IsEnd            bool   `json:"is_end"`
	IsTruncated      bool   `json:"is_truncated"`
	Result           string `json:"result"`
	NeedClearHistory bool   `json:"need_clear_history"`
	FinishReason     string `json:"finish_reason"`
	Usage            struct {
		PromptTokens     int64 `json:"prompt_tokens"`
		CompletionTokens int64 `json:"completion_tokens"`
		TotalTokens      int64 `json:"total_tokens"`
	} `json:"usage"`
	ErrorCode int64  `json:"error_code"`
	ErrorMsg  string `json:"error_msg"`
}

type BaiDuErrorInfo struct {
	ErrorCode int64  `json:"error_code"`
	ErrorMsg  string `json:"error_msg"`
}

func (b *BaiDuServer) ChatStream(requestPath string, data []byte, msgCh chan string, errChan chan error) (*Response, error) {
	ret := &Response{RequestData: data, ResponseData: make([]byte, 0)}

	token, err := b.Token()
	if err != nil {
		return ret, err
	}
	requestPath = requestPath + "?access_token=" + token

	headers := map[string]string{"Content-Type": "application/json"}
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
		line, _, err := reader.ReadLine()
		ret.ResponseData = append(ret.ResponseData, line...)
		line = bytes.TrimSuffix(line, []byte("\n"))

		if err != nil {

			if errors.Is(err, io.EOF) {
				resErr := err
				errStruct := &BaiDuErrorInfo{}
				if err := json.Unmarshal(line, errStruct); err == nil {
					resErr = errors.New(errStruct.ErrorMsg)
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

		retStruct := BaiDuStreamResp{}
		if err := json.Unmarshal(line, &retStruct); err != nil {
			errChan <- err
			return ret, err
		}

		if retStruct.ErrorCode != 0 {
			err := errors.New(retStruct.ErrorMsg)
			errChan <- err
			return ret, err
		}

		if retStruct.IsEnd {
			ret.PromptTokens = retStruct.Usage.PromptTokens
			ret.CompletionTokens = retStruct.Usage.CompletionTokens
			close(msgCh)
			break
		}

		ret.ResponseText += retStruct.Result
		msgCh <- retStruct.Result
	}

	return ret, nil
}
