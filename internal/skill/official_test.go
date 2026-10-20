package skill

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOfficialReleaseHasSixUsableBoundedMethodsAndExplicitExamples(t *testing.T) {
	packages, err := OfficialPackages()
	require.NoError(t, err)
	require.Len(t, packages, 6)
	seen, targets := map[string]bool{}, map[string]int{}
	for _, release := range packages {
		require.False(t, seen[release.Key])
		seen[release.Key] = true
		p, err := ValidatePackage(release.Package.Files)
		require.NoError(t, err)
		require.Equal(t, release.Package, p)
		require.NotEmpty(t, p.DisplayName)
		targets[p.Target]++
		text := p.Files[0].Text
		for _, section := range []string{"适用范围", "输入", "步骤", "输出", "限制", "示例", "仅用于说明"} {
			require.Contains(t, text, section)
		}
		if p.Target == ImageTarget {
			require.Contains(t, text, "不增加任务")
			require.False(t, p.Supports(TitleTarget))
			var brief string
			for _, f := range p.Files {
				if f.Path == "references/visual-brief.md" {
					brief = f.Text
				}
			}
			require.NotEmpty(t, brief, "each current official image release needs its own visual resource")
		}
		require.False(t, strings.Contains(text, "TODO"))
	}
	require.Equal(t, map[string]int{TitleTarget: 3, ImageTarget: 3}, targets)
}
