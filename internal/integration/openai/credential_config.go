package openai

import (
	"time"
)

type ClientConfig struct {
	APIKey   string
	Model    string
	BaseURL  string
	APIStyle string
	Timeout  time.Duration
}
