package description

import (
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestValidateRichDescription(t *testing.T) {
	image := "https://images.example.org/a.png"
	for _, value := range []string{
		"", "<p>Hello &amp; &lt;world&gt;</p>", "<h2>Title</h2><p><strong>Bold</strong><em>Italic</em><s>old</s><br></p>",
		"<ul><li>one</li><li>two</li></ul><blockquote><p>quote</p></blockquote>",
		"<p><a href=\"https://example.org/path?q=1&amp;x=2\" target=\"_blank\" rel=\"noopener noreferrer nofollow\">link</a></p>",
		"<img src=\"" + image + "\" alt=\"pic\">",
	} {
		require.NoError(t, Validate(value, HTML, []string{image}), value)
	}
	for _, value := range []string{
		"<script>alert(1)</script><p>safe</p>", "<p onclick=\"alert(1)\">x</p>", "<svg><a>x</a></svg>",
		"<img src=\"https://other.example.org/a.png\">", "<img src=\"data:image/png;base64,AAAA\">",
		"<a href=\"javascript:alert(1)\">x</a>", "<a href=\"//example.org\">x</a>",
		"<a href=\"https://user:pass@example.org\">x</a>", "<a href=\"https://example.org/#x\">x</a>",
		"<a href=\"https://example.org\" target=\"_blank\">x</a>",
		"<p style=\"color:red\">x</p>", "<!-- comment --><p>x</p>", "<html><body>x</body></html>",
		strings.Repeat("<blockquote>", 33) + "x" + strings.Repeat("</blockquote>", 33),
		strings.Repeat("<p>x</p>", 1025),
	} {
		require.Error(t, Validate(value, HTML, []string{image}), value)
	}
	require.NoError(t, Validate("<script>literal</script>", "", nil))
	require.Error(t, Validate("x", "html", nil))
	require.Error(t, Validate(strings.Repeat("x", 32769), HTML, nil))
}
func TestPlainTextProjection(t *testing.T) {
	require.Equal(t, "<b>literal</b>\n ", PlainText("<b>literal</b>\n ", ""))
	require.Equal(t, "Heading\nOne & two\nline\nFirst\nSecond\nlink", PlainText("<h2>Heading</h2><p>One &amp; two<br>line</p><ul><li>First</li><li>Second</li></ul><p><a href=\"https://example.org\">link</a><img src=\"https://example.org/a.png\" alt=\"ignored\"></p>", HTML))
}

func TestProjectionUnicodeWhitespace(t *testing.T) {
	require.Equal(t, "x", PlainText("<p>\u0085x\u0085</p>", HTML))
	require.Equal(t, "\uFEFFx\uFEFF", PlainText("<p>\uFEFFx\uFEFF</p>", HTML))
}
