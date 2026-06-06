package config

type Config struct {
	LLM LLM `mapstructure:"llm"`
}

type LLM struct {
	Provider string `mapstructure:"provider"`
	Model    string `mapstructure:"model"`
	APIKey   string `mapstructure:"api_key"`
	BaseURL  string `mapstructure:"base_url"`
}
