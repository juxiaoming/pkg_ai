package pkg_ai

import "errors"

var (
	config  Config // 全局配置
	hasInit bool   // 配置是否初始化
)

type Config struct {
}

type RequestData struct {
	Supplier    string      `json:"supplier"`
	Model       string      `json:"model"`
	UserQuery   string      `json:"user_query"`
	SystemQuery string      `json:"system_query"`
	History     [][2]string `json:"history"`
	Temperature float64     `json:"temperature"`
}

type ResponseData struct {
	OriginalData     []byte `json:"original_data"`
	PromptTokens     int64  `json:"prompt_tokens"`
	CompletionTokens int64  `json:"completion_tokens"`
	Response         string `json:"response"`
}

type Ability interface {
	build(data RequestData) ([]byte, error)
	streamBuild(data RequestData) ([]byte, error)
	Chat(data []byte) (*ResponseData, error)
	ChatStream(data []byte, msgCh chan string, errChan chan error, stopChan chan struct{}) (*ResponseData, error)
}

func Init(conf *Config) {
	config = *conf
	hasInit = true
}

type Server struct {
	client      Ability
	ImplementId int8 `json:"implement_id"`
}

func NewServer(implementId int8) (*Server, error) {
	if !hasInit {
		return nil, errors.New("配置未初始化,请先调用【Init】方法")
	}
	var client Ability

	return &Server{client: client, ImplementId: implementId}, nil
}

// Chat 阻塞式对话
func (s *Server) Chat(data RequestData) (*ResponseData, error) {
	payload, err := s.client.build(data)
	if err != nil {
		return nil, err
	}

	return s.client.Chat(payload)
}

// ChatStream 流式对话
func (s *Server) ChatStream(data RequestData, msgCh chan string, errChan chan error, stopChan chan struct{}) (*ResponseData, error) {
	payload, err := s.client.streamBuild(data)
	if err != nil {
		return nil, err
	}

	return s.client.ChatStream(payload, msgCh, errChan, stopChan)
}
