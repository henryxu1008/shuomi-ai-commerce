package jsonx

import (
	"encoding/json"
	"fmt"
)

// UnmarshalString 从JSON字符串解析到目标对象
// 这是一个泛型辅助函数，用于减少重复的JSON解析代码
func UnmarshalString[T any](jsonStr string, target *T, errorPrefix string) error {
	if err := json.Unmarshal([]byte(jsonStr), target); err != nil {
		if errorPrefix != "" {
			return fmt.Errorf("%s: %w", errorPrefix, err)
		}
		return fmt.Errorf("JSON解析失败: %w", err)
	}
	return nil
}
