package pkg_ai

import "errors"

var (
	config  Config // 全局配置
	hasInit bool   // 配置是否初始化
)

type Config struct {
	MoonshotKey string `json:"moonshot_key"`
}

type RequestData struct {
	Model            string      `json:"model"`                       // Model ID
	UserQuery        string      `json:"user_query"`                  // 用户提示词
	SystemQuery      string      `json:"system_query,omitempty"`      // 系统提示词
	History          [][2]string `json:"history,omitempty"`           // 历史对话
	MaxTokens        int64       `json:"max_tokens,omitempty"`        // 聊天完成时生成的最大 token 数
	Temperature      float64     `json:"temperature,omitempty"`       // 使用什么采样温度
	TopP             float64     `json:"top_p,omitempty"`             // 另一种采样方法
	N                int64       `json:"n,omitempty"`                 // 为每条输入消息生成多少个结果
	PresencePenalty  float64     `json:"presence_penalty,omitempty"`  // 存在惩罚
	FrequencyPenalty float64     `json:"frequency_penalty,omitempty"` // 频率惩罚
	ResponseFormat   string      `json:"response_format,omitempty"`   // 响应格式【text 、 json_object】
	Stop             []string    `json:"stop,omitempty"`              // 停止词
}

type Response struct {
	RequestData      []byte `json:"request_data"`      // 请求原始数据
	ResponseData     []byte `json:"response_data"`     // 响应原始数据
	PromptTokens     int64  `json:"prompt_tokens"`     // 输入提示词token
	CompletionTokens int64  `json:"completion_tokens"` // 响应token
	ResponseText     string `json:"response_text"`     // 整理后的响应结果
}

type Ability interface {
	build(data RequestData, isStream bool) ([]byte, error)
	Chat(requestPath string, data []byte) (*Response, error)
	ChatStream(requestPath string, data []byte, msgCh chan string, errChan chan error, stopChan <-chan struct{}) (*Response, error)
	Supplier() string
	RequestPath(model string) string
}

func Init(conf *Config) {
	config = *conf
	hasInit = true
}

type Server struct {
	client      Ability
	ImplementId int8 `json:"implement_id"`
}

const (
	ImplementMoonshot int8 = 1 // 月之暗面
)

func NewServer(implementId int8) (*Server, error) {
	if !hasInit {
		return nil, errors.New("配置未初始化,请先调用【Init】方法")
	}

	var client Ability

	switch implementId {
	case ImplementMoonshot:
		if len(config.MoonshotKey) == 0 {
			return nil, errors.New("缺失配置")
		}

		client = newMoonshotServer(config.MoonshotKey)
	default:
		return nil, errors.New("未定义实现")
	}

	return &Server{client: client, ImplementId: implementId}, nil
}

// Chat 阻塞式对话
func (s *Server) Chat(data RequestData) (*Response, error) {
	payload, err := s.client.build(data, false)
	if err != nil {
		return nil, err
	}

	requestPath := s.client.RequestPath(data.Model)

	return s.client.Chat(requestPath, payload)
}

// ChatStream 流式对话
func (s *Server) ChatStream(data RequestData, msgCh chan string, errChan chan error, stopChan chan struct{}) (*Response, error) {
	payload, err := s.client.build(data, true)
	if err != nil {
		return nil, err
	}

	requestPath := s.client.RequestPath(data.Model)

	return s.client.ChatStream(requestPath, payload, msgCh, errChan, stopChan)
}
