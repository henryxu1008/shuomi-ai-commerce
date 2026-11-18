package einomodel

import (
	"encoding/json"
	"errors"
	"mime"
	"net/http"
	"strings"

	strict "sigs.k8s.io/json"
)

var errOpenAIParameterRejected = errors.New("OpenAI rejected the skill authoring parameter before inference")

// This is the synchronous response to the guarded request, before the SDK
// reduces every non-success response to an error with unobserved usage.
func (g *GuardedClient) observeOpenAIParameterRejection(req *http.Request, sent []byte, response *http.Response, raw []byte) {
	if !g.cfg.skillAuthoringDirectOpenAI || req.URL.EscapedPath() != "/v1/chat/completions" || response.StatusCode != http.StatusBadRequest {
		return
	}
	mediaType, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	requestIDs := response.Header.Values("X-Request-Id")
	if err != nil || mediaType != "application/json" || len(requestIDs) != 1 || !validOpenAIRequestID(requestIDs[0]) {
		return
	}

	var wire map[string]json.RawMessage
	var maxTokens int
	if json.Unmarshal(sent, &wire) != nil || json.Unmarshal(wire["max_tokens"], &maxTokens) != nil || maxTokens <= 0 || wire["max_completion_tokens"] != nil {
		return
	}
	var rejection struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    string `json:"code"`
			Param   string `json:"param"`
		} `json:"error"`
	}
	unexpected, err := strict.UnmarshalStrict(raw, &rejection, strict.DisallowDuplicateFields, strict.DisallowUnknownFields)
	if err != nil || len(unexpected) != 0 || rejection.Error.Type != "invalid_request_error" ||
		rejection.Error.Code != "unsupported_parameter" || rejection.Error.Param != "max_tokens" {
		return
	}

	g.mu.Lock()
	g.openAIParameterRejectionIDHash = diagnosticHash(requestIDs[0])
	g.mu.Unlock()
}

func validOpenAIRequestID(value string) bool {
	id, found := strings.CutPrefix(value, "req_")
	return found && len(id) > 0 && len(value) <= 128 && !strings.ContainsFunc(id, func(r rune) bool {
		return !(r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r == '_' || r == '-')
	})
}

func (g *GuardedClient) openAIParameterRejection() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.openAIParameterRejectionIDHash
}
