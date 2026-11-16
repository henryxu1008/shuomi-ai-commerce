package skill

import (
	strict "sigs.k8s.io/json"
	"strings"
)

const ScriptPolicyVersion = "skill-script-method-v1"
const ScriptProfile = "python-title-json-v1"

type ExecutionManifest struct {
	SchemaVersion  string `json:"schemaVersion" yaml:"schemaVersion"`
	Profile        string `json:"profile" yaml:"profile"`
	Entrypoint     string `json:"entrypoint" yaml:"entrypoint"`
	Phase          string `json:"phase" yaml:"phase"`
	InputContract  string `json:"inputContract" yaml:"inputContract"`
	OutputContract string `json:"outputContract" yaml:"outputContract"`
}

func (p *Package) validateExecution() error {
	var script, manifest bool
	var spec ExecutionManifest
	for _, f := range p.Files {
		if strings.HasPrefix(f.Path, "scripts/") {
			script = true
		}
		if f.Path != "execution.json" {
			continue
		}
		manifest = true
		// Reuse the current configuration owner's strict JSON dependency.
		violations, err := strict.UnmarshalStrict([]byte(f.Text), &spec)
		if err != nil || len(violations) != 0 {
			return ErrInvalid
		}
	}
	if !script && !manifest {
		return nil
	}
	if !script || !manifest || p.Target != TitleTarget || spec.SchemaVersion != "skill-script-v1" || spec.Profile != ScriptProfile || spec.Phase != "title-candidate-postprocess" || spec.InputContract != "title-candidate-v1" || spec.OutputContract != "title-result-v1" || !strings.HasPrefix(spec.Entrypoint, "scripts/") {
		return ErrInvalid
	}
	found := false
	for _, f := range p.Files {
		if f.Path == spec.Entrypoint {
			found = true
		}
	}
	if !found {
		return ErrInvalid
	}
	p.Execution = &spec
	return nil
}

// ModelMaterial preserves frozen identity while projecting only method text.
// Saved code and the runtime manifest never become model instructions.
func (p Package) ModelMaterial() Package {
	p.Files = append([]File(nil), p.Files...)
	files := p.Files[:0]
	for _, f := range p.Files {
		if f.Path != "execution.json" && !strings.HasPrefix(f.Path, "scripts/") {
			files = append(files, f)
		}
	}
	p.Files = files
	p.Execution = nil
	return p
}
