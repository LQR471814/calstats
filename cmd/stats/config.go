package main

type Config []SourceConfig

type SourceConfig struct {
	Server    ServerConfig `json:"server"`
	Calendars []string     `json:"calendars"`
}

type ServerConfig struct {
	Url      string `json:"url"`
	Insecure bool   `json:"insecure"`
	Username string `json:"username"`
	Password string `json:"password"`
}
