package skill

import (
	"context"
	"embed"
	"io/fs"
	"strings"
	"task-processor/internal/agent"

	"github.com/google/uuid"
)

const AuthoringReleaseKey = "official-method-v1:skill-authoring"
const GeneralAuthoringVersion = 3

func OfficialReleaseID(key string) string {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte("skill:"+key)).String()
}

// Authoring is the one platform release admitted to this consumer.
func (s *Service) Authoring(ctx context.Context, version uint64) (Version, error) {
	v, _, err := s.Read(ctx, "official", OfficialReleaseID(AuthoringReleaseKey), version)
	if err != nil {
		return Version{}, err
	}
	if !v.Package.Supports(AuthoringTarget) || v.State != Enabled || v.Visibility != Shared ||
		v.Metadata.Owner != (Owner{Kind: PlatformOwner, ID: PlatformID}) {
		return Version{}, ErrNotEnabled
	}
	return v, nil
}

// Release material is installed explicitly into the existing Skill owner.
// Reads and execution never fall back to this embedded bundle.
//
//go:embed official/*.md official/visual/*.md official/authoring/*.md
var officialFiles embed.FS

type OfficialPackage struct {
	Key      string
	Version  uint64
	Package  Package
	Previous *Package
	Base     *Package
}

func OfficialPackages() ([]OfficialPackage, error) {
	entries, err := fs.ReadDir(officialFiles, "official")
	if err != nil {
		return nil, err
	}
	out := make([]OfficialPackage, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := officialFiles.ReadFile("official/" + entry.Name())
		if err != nil {
			return nil, err
		}
		p, err := ValidatePackage([]File{{Path: "SKILL.md", Text: string(data)}})
		if err != nil {
			return nil, err
		}
		release := OfficialPackage{Key: "official-method-v1:" + strings.TrimSuffix(entry.Name(), ".md"), Version: 1, Package: p}
		if release.Key == AuthoringReleaseKey {
			updated, err := officialFiles.ReadFile("official/authoring/skill-authoring.md")
			if err != nil {
				return nil, err
			}
			current, err := ValidatePackage([]File{{Path: "SKILL.md", Text: string(updated)}})
			if err != nil || !current.Supports(AuthoringTarget) {
				return nil, ErrInvalid
			}
			general, err := officialFiles.ReadFile("official/authoring/general.md")
			if err != nil {
				return nil, err
			}
			latest, err := ValidatePackage([]File{{Path: "SKILL.md", Text: string(general)}})
			if err != nil || !latest.Supports(AuthoringTarget) {
				return nil, ErrInvalid
			}
			release.Version, release.Base, release.Previous, release.Package = GeneralAuthoringVersion, &p, &current, latest
		}
		if p.Target == ImageTarget {
			brief, err := officialFiles.ReadFile("official/visual/" + entry.Name())
			if err != nil {
				return nil, err
			}
			current, err := ValidatePackage(append(append([]File(nil), p.Files...), File{Path: ImageVisualBriefPath, Text: string(brief)}))
			if err != nil {
				return nil, err
			}
			if _, err = ImageVisualBrief(current); err != nil {
				return nil, err
			}
			release.Version, release.Previous, release.Package = 2, &p, current
		}
		out = append(out, release)
	}
	return out, nil
}

type OfficialInstallResult struct {
	Installed []agent.SkillVersionRef `json:"installed"`
	Existing  []agent.SkillVersionRef `json:"existing"`
}

type OfficialInstaller interface {
	InstallOfficial(context.Context, Principal, ResolvePrincipal) (OfficialInstallResult, error)
}
