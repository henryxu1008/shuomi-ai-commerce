package config

func ValidatePlatformsConfig(platforms *PlatformsConfig) []error {
	var errors []error
	if platforms == nil {
		return errors
	}

	if platforms.Alibaba1688.Enabled {
		if platforms.Alibaba1688.Timeout <= 0 {
			errors = append(errors, &ValidationError{
				Field:   "platforms.alibaba1688.timeout",
				Message: "1688 timeout must be greater than 0",
			})
		}
		if platforms.Alibaba1688.PoolSize <= 0 {
			errors = append(errors, &ValidationError{
				Field:   "platforms.alibaba1688.poolSize",
				Message: "1688 poolSize must be greater than 0",
			})
		}
	}

	return errors
}
