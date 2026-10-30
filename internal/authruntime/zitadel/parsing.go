package zitadel

import (
	"encoding/json"
	"sort"
	"strings"
)

func ParseRolesForProject(data []byte, projectID string) []string {
	return parseRoles(data, strings.TrimSpace(projectID))
}

func parseRoles(data []byte, projectID string) []string {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}

	seen := map[string]struct{}{}
	roles := make([]string, 0, 4)
	add := func(value string) {
		role := strings.TrimSpace(value)
		if role == "" {
			return
		}
		if _, ok := seen[role]; ok {
			return
		}
		seen[role] = struct{}{}
		roles = append(roles, role)
	}

	keys := make([]string, 0, 4)
	if projectID != "" {
		keys = append(keys, "urn:zitadel:iam:org:project:"+projectID+":roles")
	}
	keys = append(keys, "urn:zitadel:iam:org:project:roles", "roles", "role")
	for _, key := range keys {
		value, ok := raw[key]
		if !ok {
			continue
		}

		var claimValue any
		if err := json.Unmarshal(value, &claimValue); err == nil {
			addRoleClaimValue(claimValue, add)
		}
	}

	return roles
}

func addRoleClaimValue(value any, add func(string)) {
	switch typed := value.(type) {
	case string:
		for _, role := range strings.Split(typed, ",") {
			add(role)
		}
	case []any:
		for _, item := range typed {
			addRoleClaimValue(item, add)
		}
	case map[string]any:
		roles := make([]string, 0, len(typed))
		for role := range typed {
			roles = append(roles, role)
		}
		sort.Strings(roles)
		for _, role := range roles {
			add(role)
		}
	}
}
