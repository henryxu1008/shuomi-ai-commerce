package official

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestEmbeddedGuideUsesProductProvenanceV2(t *testing.T) {
	c, err := NewEmbeddedCatalog()
	require.NoError(t, err)
	require.Equal(t, "2", c.List()[0].Revision)
	a, err := c.Get("ai-commerce-guide", "2")
	require.NoError(t, err)
	data, err := json.Marshal(a)
	require.NoError(t, err)
	require.NotContains(t, string(data), "github.com")
	var projection map[string]any
	require.NoError(t, json.Unmarshal(data, &projection))
	for _, source := range projection["sources"].([]any) {
		require.Equal(t, 1, len(source.(map[string]any)))
		require.NotEmpty(t, source.(map[string]any)["title"])
	}
	require.Contains(t, a.Body, "本指南 v2")
	require.Equal(t, "2026-10-22", a.UpdatedAt)
	_, err = c.Get("ai-commerce-guide", "1")
	require.ErrorIs(t, err, ErrNotFound)
}

func TestCatalogAcceptsNamedProvenanceWithoutURLs(t *testing.T) {
	a := article("2", "text")
	a.Sources = []Source{{Title: "企业知识与引用规则"}}
	_, err := NewCatalog([]Article{a})
	require.NoError(t, err)
}

func article(revision, body string) Article {
	return Article{Summary: Summary{ID: "guide", Revision: revision, Title: "Guide", Summary: "How to use", Category: "AI电商应用指南", UpdatedAt: "2026-10-22", MaintainedBy: "硕米", Digest: Digest(body)}, Body: body, Sources: []Source{{Title: "Source"}}}
}
func TestCatalogExactVersionAndImmutableCopies(t *testing.T) {
	original := article("1", "original")
	next := article("2", "next")
	c, err := NewCatalog([]Article{original, next})
	require.NoError(t, err)
	original.Sources[0].Title = "changed"
	require.Equal(t, "2", c.List()[0].Revision)
	got, err := c.Get("guide", "1")
	require.NoError(t, err)
	require.Equal(t, "original", got.Body)
	require.Equal(t, "Source", got.Sources[0].Title)
	got.Sources[0].Title = "mutated result"
	again, err := c.Get("guide", "1")
	require.NoError(t, err)
	require.Equal(t, "Source", again.Sources[0].Title)
	_, err = c.Get("guide", "3")
	require.ErrorIs(t, err, ErrNotFound)
	_, err = c.Get("guide", "01")
	require.ErrorIs(t, err, ErrInvalid)
}
func TestCatalogRejectsCorruptionAndMalformedSourceNames(t *testing.T) {
	for _, modify := range []func(*Article){
		func(a *Article) { a.Digest = strings.Repeat("0", 64) },
		func(a *Article) { a.Body = strings.Repeat("x", MaxBodyBytes+1); a.Digest = Digest(a.Body) },
		func(a *Article) { a.Revision = "01" }, func(a *Article) { a.ID = "../secret" },
		func(a *Article) { a.Sources[0].Title = "" },
		func(a *Article) { a.Sources[0].Title = strings.Repeat("x", 513) },
		func(a *Article) { a.Sources[0].Title = "bad\x00name" },
		func(a *Article) { a.Sources[0].Title = string([]byte{0xff}) },
		func(a *Article) { a.Sources = nil },
		func(a *Article) { a.Sources = make([]Source, 11) },
		func(a *Article) { a.Body = ""; a.Digest = Digest(a.Body) },
	} {
		a := article("1", "text")
		modify(&a)
		_, err := NewCatalog([]Article{a})
		require.Error(t, err)
	}
	a := article("1", "text")
	_, err := NewCatalog([]Article{a, a})
	require.Error(t, err)
}
func TestEmbeddedGuideIsRealAndVersionBound(t *testing.T) {
	c, err := NewEmbeddedCatalog()
	require.NoError(t, err)
	require.Len(t, c.List(), 1)
	summary := c.List()[0]
	a, err := c.Get(summary.ID, summary.Revision)
	require.NoError(t, err)
	require.Contains(t, a.Body, "Human Review")
	require.Contains(t, a.Body, "默认")
	require.Contains(t, a.Body, "AI引用尚未开放")
	require.Equal(t, a.Digest, Digest(a.Body))
	require.NotEmpty(t, a.Sources)
}
