package aiidentity

import (
	"context"
	"slices"
	"strings"
)

type requestRolesContextKey struct{}

func RequestUserIDFromContext(ctx context.Context) string { return FromContext(ctx).UserID }

func WithRequestRoles(ctx context.Context, roles []string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	normalized := normalizeRequestRoles(roles)
	if len(normalized) == 0 {
		return ctx
	}
	return context.WithValue(ctx, requestRolesContextKey{}, normalized)
}

func RequestRolesFromContext(ctx context.Context) []string {
	if ctx == nil {
		return nil
	}
	roles, ok := ctx.Value(requestRolesContextKey{}).([]string)
	if !ok || len(roles) == 0 {
		return nil
	}
	return append([]string(nil), roles...)
}

func normalizeRequestRoles(roles []string) []string {
	out := make([]string, 0, len(roles))
	for _, role := range roles {
		normalized := strings.TrimSpace(role)
		if normalized == "" || slices.Contains(out, normalized) {
			continue
		}
		out = append(out, normalized)
	}
	return out
}
