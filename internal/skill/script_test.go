package skill

import (
	"strings"
	"testing"
)

func scriptFiles() []File {
	return []File{{Path: "SKILL.md", Text: "---\nname: title-script\ndescription: Format an existing candidate\nmetadata:\n  listingkit-method-v1: product-title\n---\nUse the saved formatting script.\n"},
		{Path: "execution.json", Text: `{"schemaVersion":"skill-script-v1","profile":"python-title-json-v1","entrypoint":"scripts/main.py","phase":"title-candidate-postprocess","inputContract":"title-candidate-v1","outputContract":"title-result-v1"}`},
		{Path: "scripts/main.py", Text: "import json\nprint('synthetic')\n"}}
}

func TestScriptPackageRequiresExplicitCompleteContract(t *testing.T) {
	files := scriptFiles()
	p, err := ValidatePackage(files)
	if err != nil {
		t.Fatalf("explicit saved script package rejected: %v", err)
	}
	if (Version{Package: p}).Ref().PolicyVersion != "skill-script-method-v1" {
		t.Fatal("script mislabeled as text")
	}
	material := p.ModelMaterial()
	if material.Digest != p.Digest || material.Execution != nil || len(material.Files) != 1 || material.Files[0].Path != "SKILL.md" {
		t.Fatal("model received code/runtime manifest or lost frozen identity")
	}
	original := p.Digest
	files[2].Text += "# changed\n"
	p, err = ValidatePackage(files)
	if err != nil || p.Digest == original {
		t.Fatal("script not bound to digest", err)
	}
	for _, bad := range []struct {
		name  string
		files []File
	}{
		{"script without manifest", append(scriptFiles()[:1], scriptFiles()[2])},
		{"manifest without script", scriptFiles()[:2]},
		{"unmarked", func() []File {
			f := scriptFiles()
			f[0].Text = strings.ReplaceAll(f[0].Text, "product-title", "unknown")
			return f
		}()},
		{"unknown field", func() []File {
			f := scriptFiles()
			f[1].Text = strings.TrimSuffix(f[1].Text, "}") + `,"network":"allow"}`
			return f
		}()},
		{"duplicate field", func() []File {
			f := scriptFiles()
			f[1].Text = strings.TrimSuffix(f[1].Text, "}") + `,"profile":"python-title-json-v1"}`
			return f
		}()},
		{"wrong profile", func() []File {
			f := scriptFiles()
			f[1].Text = strings.ReplaceAll(f[1].Text, "python-title-json-v1", "shell-v1")
			return f
		}()},
	} {
		t.Run(bad.name, func(t *testing.T) {
			if _, err := ValidatePackage(bad.files); err == nil {
				t.Fatal("unsupported script package admitted")
			}
		})
	}
}
