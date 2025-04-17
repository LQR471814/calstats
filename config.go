package main

type ServerConfig struct {
	Url      string `json:"url"`
	Insecure bool   `json:"insecure"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type SourceConfig struct {
	Server    ServerConfig `json:"server"`
	Calendars []string     `json:"calendars"`
}

type Config []SourceConfig
