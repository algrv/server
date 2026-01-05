package config

type Config struct {
	OpenAIKey          string
	AnthropicKey       string
	SupabaseConnString string
	RedisURL           string
	Environment        string
}

type Flags struct {
	Path  string
	Clear bool
}
