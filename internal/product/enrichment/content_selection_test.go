package enrichment

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestContentSelectionIsCanonicalAndComparable(t *testing.T) {
	s, err := NewContentSelection("en-US", []string{"seo_keywords", "title", "selling_points", "description"})
	require.NoError(t, err)
	require.True(t, s.Valid())
	require.False(t, s.Absent())
	require.Equal(t, []string{"title", "description", "selling_points", "seo_keywords"}, s.Fields())
	reordered, err := NewContentSelection("en-US", []string{"title", "description", "selling_points", "seo_keywords"})
	require.NoError(t, err)
	require.True(t, reordered == s)
	require.True(t, (ContentSelection{}).Absent())
	require.False(t, (ContentSelection{}).Valid())
	for _, locale := range []string{"", "und", "und-US", "en_US", "EN-us", " en-US", "invalid-locale-tag"} {
		_, err := NewContentSelection(locale, []string{"title"})
		require.ErrorIs(t, err, ErrInputInvalid, locale)
	}
	for _, fields := range [][]string{nil, {}, {"title", "title"}, {"price"}, {"Title"}, {"description", "stock"}} {
		_, err := NewContentSelection("zh-Hans", fields)
		require.ErrorIs(t, err, ErrInputInvalid, fields)
	}
}

func TestContentSelectionFreezesLanguageAndWholeFieldSet(t *testing.T) {
	a, err := NewContentSelection("en", []string{"title"})
	require.NoError(t, err)
	b, err := NewContentSelection("de", []string{"title"})
	require.NoError(t, err)
	c, err := NewContentSelection("en", []string{"title", "description"})
	require.NoError(t, err)
	require.NotEqual(t, a, b)
	require.NotEqual(t, a, c)
	require.True(t, a.TitleOnly())
	require.False(t, c.TitleOnly())
}
