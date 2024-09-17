package pkg_ai

import "errors"

var (
	config  Config // 全局配置
	hasInit bool   // 配置是否初始化
)

type Config struct {
	MoonshotUrl           string `json:"moonshot_url"`
	MoonshotKey           string `json:"moonshot_key"`
	MinimaxiUrl           string `json:"minimaxi_url"`
	MinimaxiKey           string `json:"minimaxi_key"`
	VolcUrl               string `json:"volc_url"`
	VolcKey               string `json:"volc_key"`
	BaiDuUrl              string `json:"bai_du_url"`
	BaiDuClientId         string `json:"bai_du_client_id"`
	BaiDuClientSecret     string `json:"bai_du_client_secret"`
	QwenUrl               string `json:"qwen_url"`
	QwenKey               string `json:"qwen_key"`
	HunyuanUrl            string `json:"hunyuan_url"`
	HunyuanClientId       string `json:"hunyuan_client_id"`
	HunyuanClientSecret   string `json:"hunyuan_client_secret"`
	GlmUrl                string `json:"glm_url"`
	GlmKey                string `json:"glm_key"`
	XfYunUrl              string `json:"xf_yun_url"`
	XfYunKey              string `json:"xf_yun_key"`
	BaiChuanUrl           string `json:"bai_chuan_url"`
	BaiChuanKey           string `json:"bai_chuan_key"`
	SensenovaUrl          string `json:"sensenova_url"`
	SensenovaClientId     string `json:"sensenova_client_id"`
	SensenovaClientSecret string `json:"sensenova_client_secret"`
}

type RequestData struct {
	Model             string      `json:"model"`                         // Model ID
	UserQuery         string      `json:"user_query"`                    // 用户提示词
	SystemQuery       string      `json:"system_query,omitempty"`        // 系统提示词
	History           [][2]string `json:"history,omitempty"`             // 历史对话
	MaxTokens         int64       `json:"max_tokens,omitempty"`          // 聊天完成时生成的最大 token 数
	Temperature       float64     `json:"temperature,omitempty"`         // 使用什么采样温度
	TopP              float64     `json:"top_p,omitempty"`               // 另一种采样方法
	N                 int64       `json:"n,omitempty"`                   // 为每条输入消息生成多少个结果
	PresencePenalty   float64     `json:"presence_penalty,omitempty"`    // 存在惩罚
	FrequencyPenalty  float64     `json:"frequency_penalty,omitempty"`   // 频率惩罚
	ResponseFormat    string      `json:"response_format,omitempty"`     // 响应格式【text 、 json_object】
	Stop              []string    `json:"stop,omitempty"`                // 停止词
	MaskSensitiveInfo bool        `json:"mask_sensitive_info,omitempty"` // 对输出中易涉及隐私问题的文本信息进行打码
}

type Response struct {
	RequestHeader    []byte   `json:"request_header"`    // 请求header头部信息
	RequestBody      []byte   `json:"request_body"`      // 请求body体信息
	ResponseData     [][]byte `json:"response_data"`     // 响应原始数据
	PromptTokens     int64    `json:"prompt_tokens"`     // 输入提示词token
	CompletionTokens int64    `json:"completion_tokens"` // 响应token
	ResponseText     string   `json:"response_text"`     // 整理后的响应结果
}

type Ability interface {
	build(data RequestData, isStream bool) ([]byte, error)
	Chat(requestPath string, data []byte) (*Response, error)
	ChatStream(requestPath string, data []byte, msgCh chan string, errChan chan error) (*Response, error)
	Supplier() string
	RequestPath() string
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
	ImplementMoonshot  int8 = 1  // 月之暗面
	ImplementMinimaxi  int8 = 2  // Minimaxi
	ImplementVolc      int8 = 3  // Volc
	ImplementBaidu     int8 = 4  // 百度
	ImplementQwen      int8 = 5  // 通义千问
	ImplementHunyuan   int8 = 6  // 混元大模型
	ImplementGlm       int8 = 7  // 智谱清言
	ImplementXfYun     int8 = 8  // 科大讯飞
	ImplementBaiChuan  int8 = 9  // 百川智能
	ImplementSensenova int8 = 10 // 商汤日日新
)

func NewServer(implementId int8) (*Server, error) {
	if !hasInit {
		return nil, errors.New("配置未初始化,请先调用【Init】方法")
	}

	var client Ability

	switch implementId {
	case ImplementMoonshot:
		if len(config.MoonshotKey) == 0 || len(config.MoonshotUrl) == 0 {
			return nil, errors.New("缺失配置")
		}

		client = newMoonshotServer(config.MoonshotUrl, config.MoonshotKey)
	case ImplementMinimaxi:
		if len(config.MinimaxiUrl) == 0 || len(config.MinimaxiKey) == 0 {
			return nil, errors.New("缺失配置")
		}

		client = newMinimaxiServer(config.MinimaxiUrl, config.MinimaxiKey)
	case ImplementVolc:
		if len(config.VolcUrl) == 0 || len(config.VolcKey) == 0 {
			return nil, errors.New("缺失配置")
		}

		client = newVolcServer(config.VolcUrl, config.VolcKey)
	case ImplementBaidu:
		if len(config.BaiDuUrl) == 0 || len(config.BaiDuClientId) == 0 || len(config.BaiDuClientSecret) == 0 {
			return nil, errors.New("缺失配置")
		}

		client = newBaiDuServer(config.BaiDuUrl, config.BaiDuClientId, config.BaiDuClientSecret)
	case ImplementQwen:
		if len(config.QwenUrl) == 0 || len(config.QwenKey) == 0 {
			return nil, errors.New("缺失配置")
		}

		client = newQwenServer(config.QwenUrl, config.QwenKey)
	case ImplementHunyuan:
		if len(config.HunyuanUrl) == 0 || len(config.HunyuanClientId) == 0 || len(config.HunyuanClientSecret) == 0 {
			return nil, errors.New("缺失配置")
		}

		client = newHunyuanServer(config.HunyuanUrl, config.HunyuanClientId, config.HunyuanClientSecret)
	case ImplementGlm:
		if len(config.GlmUrl) == 0 || len(config.GlmKey) == 0 {
			return nil, errors.New("缺失配置")
		}

		client = newQwenServer(config.GlmUrl, config.GlmKey)
	case ImplementXfYun:
		if len(config.XfYunUrl) == 0 || len(config.XfYunKey) == 0 {
			return nil, errors.New("缺失配置")
		}

		client = newXfYunServer(config.XfYunUrl, config.XfYunKey)
	case ImplementBaiChuan:
		if len(config.BaiChuanUrl) == 0 || len(config.BaiChuanKey) == 0 {
			return nil, errors.New("缺失配置")
		}

		client = newBaiChuanServer(config.BaiChuanUrl, config.BaiChuanKey)
	case ImplementSensenova:
		if len(config.SensenovaUrl) == 0 || len(config.SensenovaClientId) == 0 || len(config.SensenovaClientSecret) == 0 {
			return nil, errors.New("缺失配置")
		}

		client = newSensenovaServer(config.SensenovaUrl, config.SensenovaClientId, config.SensenovaClientSecret)
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

	return s.client.Chat(s.client.RequestPath(), payload)
}

// ChatStream 流式对话
func (s *Server) ChatStream(data RequestData, msgCh chan string, errChan chan error) (*Response, error) {
	payload, err := s.client.build(data, true)
	if err != nil {
		return nil, err
	}

	return s.client.ChatStream(s.client.RequestPath(), payload, msgCh, errChan)
}

// CustomizeChat 自定义参数阻塞式对话, 用户自己实现请求的body参数
func (s *Server) CustomizeChat(payload []byte) (*Response, error) {
	return s.client.Chat(s.client.RequestPath(), payload)
}

// CustomizeChatStream 自定义参数流式对话, 用户自己实现请求的body参数
func (s *Server) CustomizeChatStream(payload []byte, msgCh chan string, errChan chan error) (*Response, error) {
	return s.client.ChatStream(s.client.RequestPath(), payload, msgCh, errChan)
}

func (s *Server) Supplier() string {
	return s.client.Supplier()
}
