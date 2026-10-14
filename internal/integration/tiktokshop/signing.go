package tiktokshop

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"mime"
	"net/url"
	"sort"
	"strings"
)

var errSignature = errors.New("invalid TikTok Shop signature input")

// signature follows the official Go protocol example. body is the exact byte
// sequence subsequently sent; multipart bodies are excluded by media type.
func signature(path string, query url.Values, body []byte, contentType, secret string) (string, error) {
	if !strings.HasPrefix(path, "/") || strings.ContainsAny(path, "?#\r\n") || secret == "" {
		return "", errSignature
	}
	media, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return "", errSignature
	}
	keys := make([]string, 0, len(query))
	for key, values := range query {
		if len(values) != 1 {
			return "", errSignature
		}
		if key != "sign" && key != "access_token" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(secret))
	_, _ = mac.Write([]byte(path))
	for _, key := range keys {
		_, _ = mac.Write([]byte(key))
		_, _ = mac.Write([]byte(query.Get(key)))
	}
	if media != "multipart/form-data" {
		_, _ = mac.Write(body)
	}
	_, _ = mac.Write([]byte(secret))
	return hex.EncodeToString(mac.Sum(nil)), nil
}
