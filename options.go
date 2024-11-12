package pkg_ai

type Options interface {
	Apply(*Config)
}

type WithConfig func(*Config)

func (w WithConfig) Apply(c *Config) {
	w(c)
}

func WithBaiChuanConfig(url, key string) WithConfig {
	return func(c *Config) {
		c.BaiChuanUrl = url
		c.BaiChuanKey = key
	}
}

func WithBaiDuConfig(url, clientId, clientSecret string) WithConfig {
	return func(c *Config) {
		c.BaiDuUrl = url
		c.BaiDuClientId = clientId
		c.BaiDuClientSecret = clientSecret
	}
}

func WithGlmConfig(url, key string) WithConfig {
	return func(c *Config) {
		c.GlmUrl = url
		c.GlmKey = key
	}
}

func WithHunYuanConfig(url, clientId, clientSecret string) WithConfig {
	return func(c *Config) {
		c.HunyuanUrl = url
		c.HunyuanClientId = clientId
		c.HunyuanClientSecret = clientSecret
	}
}

func WithMinimaxiConfig(url, key string) WithConfig {
	return func(c *Config) {
		c.MinimaxiUrl = url
		c.MinimaxiKey = key
	}
}

func WithMoonshotConfig(url, key string) WithConfig {
	return func(c *Config) {
		c.MoonshotUrl = url
		c.MoonshotKey = key
	}
}

func WithQwenConfig(url, key string) WithConfig {
	return func(c *Config) {
		c.QwenUrl = url
		c.QwenKey = key
	}
}

func WithSensenovaConfig(url, clientId, clientSecret string) WithConfig {
	return func(c *Config) {
		c.SensenovaUrl = url
		c.SensenovaClientId = clientId
		c.SensenovaClientSecret = clientSecret
	}
}

func WithVolcConfig(url, key string) WithConfig {
	return func(c *Config) {
		c.VolcUrl = url
		c.VolcKey = key
	}
}

func WithXfYunConfig(url, key string) WithConfig {
	return func(c *Config) {
		c.XfYunUrl = url
		c.XfYunKey = key
	}
}
