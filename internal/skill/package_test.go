package skill

import (
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func titlePackage() []File {
	return []File{{Path: "SKILL.md", Text: "---\nname: product-title-normalization\ndescription: Normalize the saved product title without inventing facts.\n---\nUse only saved facts. Missing facts require confirmation.\n"}}
}

func TestValidatePackageFreezesAllSupportedMaterials(t *testing.T) {
	files := append(titlePackage(), File{Path: "references/rules.md", Text: "Do not claim unverified certification."})
	files[0].Text += "See [rules](references/rules.md).\n"
	p, err := ValidatePackage(files)
	require.NoError(t, err)
	require.Equal(t, "product-title-normalization", p.Name)
	require.Len(t, p.Digest, 64)
	require.Len(t, p.Files, 2)
	copyFiles := []File{files[1], files[0]}
	same, err := ValidatePackage(copyFiles)
	require.NoError(t, err)
	require.Equal(t, p.Digest, same.Digest)
	copyFiles[0].Text += "Changed rule."
	changed, err := ValidatePackage(copyFiles)
	require.NoError(t, err)
	require.NotEqual(t, p.Digest, changed.Digest)
	files[0].Text = "tampered after validation"
	require.Contains(t, p.Files[0].Text, "saved facts")
}

func TestValidatePackageRejectsExecutionAndEscapingMaterials(t *testing.T) {
	for _, value := range []struct {
		name  string
		files []File
	}{
		{"missing frontmatter", []File{{Path: "SKILL.md", Text: "No manifest"}}},
		{"unknown model", []File{{Path: "SKILL.md", Text: "---\nname: title\ndescription: title\nmodel: other\n---\nTitle"}}},
		{"allowed tools", []File{{Path: "SKILL.md", Text: "---\nname: title\ndescription: title\nallowed-tools: Bash\n---\nTitle"}}},
		{"duplicate field", []File{{Path: "SKILL.md", Text: "---\nname: title\nname: another\ndescription: title\n---\nTitle"}}},
		{"yaml alias", []File{{Path: "SKILL.md", Text: "---\nname: &name title\ndescription: *name\n---\nTitle"}}},
		{"script", append(titlePackage(), File{Path: "scripts/run.py", Text: "print('bad')"})},
		{"parent path", append(titlePackage(), File{Path: "references/../secret.md", Text: "bad"})},
		{"windows path", append(titlePackage(), File{Path: "C:/secret.md", Text: "bad"})},
		{"encoded path", append(titlePackage(), File{Path: "references/%2e%2e/secret.md", Text: "bad"})},
		{"case collision", append(titlePackage(), File{Path: "skill.md", Text: "bad"})},
		{"duplicate file", append(titlePackage(), titlePackage()[0])},
		{"oversize file", append(titlePackage(), File{Path: "references/large.md", Text: strings.Repeat("x", MaxFileBytes+1)})},
	} {
		t.Run(value.name, func(t *testing.T) { _, err := ValidatePackage(value.files); require.Error(t, err) })
	}
	for _, link := range []string{"[remote](https://example.invalid/rules)", "[outside](../secret.md)", "[missing](references/missing.md)", "<script>bad()</script>", "![image](references/rules.md)"} {
		t.Run(link, func(t *testing.T) {
			files := titlePackage()
			files[0].Text += "\n" + link
			_, err := ValidatePackage(files)
			require.Error(t, err)
		})
	}
}
