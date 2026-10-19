package config

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
)

type Validator struct {
	openai    *OpenAIConfig
	browser   *BrowserConfig
	amazon    *AmazonConfig
	platforms *PlatformsConfig
}

func NewValidator(
	openai *OpenAIConfig,
	browser *BrowserConfig,
	amazon *AmazonConfig,
	platforms *PlatformsConfig,
) *Validator {
	return &Validator{
		openai:    openai,
		browser:   browser,
		amazon:    amazon,
		platforms: platforms,
	}
}

func (v *Validator) Validate() []error {
	var errors []error

	errors = append(errors, ValidateOpenAIConfig(v.openai)...)
	errors = append(errors, ValidateBrowserConfig(v.browser)...)
	errors = append(errors, ValidateAmazonConfig(v.amazon)...)
	errors = append(errors, ValidatePlatformsConfig(v.platforms)...)
	return errors
}

func (v *Validator) ValidateAndLog(logger *logrus.Logger) bool {
	errors := v.Validate()
	if len(errors) == 0 {
		logger.Info("configuration validation passed")
		return true
	}

	logger.Error("configuration validation failed:")
	for _, line := range formatValidationErrors(errors) {
		logger.Error(line)
	}

	return false
}

func (v *Validator) ValidateWithError() error {
	errors := v.Validate()
	if len(errors) == 0 {
		return nil
	}

	return fmt.Errorf("config validation failed:\n%s", strings.Join(formatValidationErrors(errors), "\n"))
}
