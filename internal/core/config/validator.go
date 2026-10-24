package config

import (
	"fmt"
	"sort"
	"strings"
)

func ValidateConfig(c *Config) []error {
	var errors []error
	errors = append(errors, ValidateOpenAIConfig(&c.OpenAI)...)
	errors = append(errors, ValidateBrowserConfig(&c.Browser)...)
	errors = append(errors, ValidateAmazonConfig(&c.Amazon)...)
	errors = append(errors, ValidatePlatformsConfig(&c.Platforms)...)
	errors = append(errors, ValidateListingKitConfig(&c.ListingKit)...)
	errors = append(errors, ValidateWorkbenchConfig(&c.Workbench, &c.ListingKit.Zitadel)...)
	return errors
}

func ValidateConfigWithError(c *Config) error {
	errors := ValidateConfig(c)
	if len(errors) == 0 {
		return nil
	}

	return fmt.Errorf("config validation failed:\n%s", strings.Join(formatValidationErrors(errors), "\n"))
}

func formatValidationErrors(errors []error) []string {
	grouped := make(map[string][]string)
	var order []string

	for _, err := range errors {
		module, message := classifyValidationError(err)
		if _, exists := grouped[module]; !exists {
			order = append(order, module)
		}
		grouped[module] = append(grouped[module], message)
	}

	sort.SliceStable(order, func(i, j int) bool {
		return validationModuleRank(order[i]) < validationModuleRank(order[j])
	})

	var lines []string
	for _, module := range order {
		lines = append(lines, fmt.Sprintf("[%s]", module))
		for _, message := range grouped[module] {
			lines = append(lines, "  - "+message)
		}
	}

	return lines
}

func classifyValidationError(err error) (string, string) {
	if ve, ok := err.(*ValidationError); ok {
		message := fmt.Sprintf("%s: %s", ve.Field, ve.Message)
		if ve.Hint != "" {
			message = fmt.Sprintf("%s (hint: %s)", message, ve.Hint)
		}
		return moduleFromField(ve.Field), message
	}
	return "General", err.Error()
}

func moduleFromField(field string) string {
	switch {
	case strings.HasPrefix(field, "openai."):
		return "OpenAI"
	case strings.HasPrefix(field, "browser."):
		return "Browser"
	case strings.HasPrefix(field, "amazon.spapi."):
		return "Amazon SP-API"
	case strings.HasPrefix(field, "amazon."):
		return "Amazon"
	case strings.HasPrefix(field, "platforms.temu."):
		return "Platforms.TEMU"
	case strings.HasPrefix(field, "platforms.shein."):
		return "Platforms.SHEIN"
	case strings.HasPrefix(field, "platforms.alibaba1688."):
		return "Platforms.1688"
	default:
		return "General"
	}
}

func validationModuleRank(module string) int {
	switch module {
	case "OpenAI":
		return 1
	case "Browser":
		return 2
	case "Amazon SP-API":
		return 3
	case "Amazon":
		return 4
	case "Platforms.TEMU":
		return 6
	case "Platforms.SHEIN":
		return 7
	case "Platforms.1688":
		return 8
	default:
		return 99
	}
}
