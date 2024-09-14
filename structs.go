package pkg_ai

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type HunyuanMessage struct {
	Role    string `json:"Role"`
	Content string `json:"Content"`
}

type StreamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}
