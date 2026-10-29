package tests

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

type imageAgentWorkloadManifest struct {
	Kind     string `yaml:"kind"`
	Metadata struct {
		Name string `yaml:"name"`
	} `yaml:"metadata"`
	Spec struct {
		Replicas int `yaml:"replicas"`
		Selector struct {
			MatchLabels map[string]string `yaml:"matchLabels"`
		} `yaml:"selector"`
		ActiveDeadlineSeconds int `yaml:"activeDeadlineSeconds"`
		BackoffLimit          int `yaml:"backoffLimit"`
		Template              struct {
			Metadata struct {
				Labels map[string]string `yaml:"labels"`
			} `yaml:"metadata"`
			Spec struct {
				RestartPolicy  string                        `yaml:"restartPolicy"`
				InitContainers []imageAgentWorkloadContainer `yaml:"initContainers"`
				Containers     []imageAgentWorkloadContainer `yaml:"containers"`
			} `yaml:"spec"`
		} `yaml:"template"`
	} `yaml:"spec"`
}

type imageAgentWorkloadContainer struct {
	Name    string   `yaml:"name"`
	Image   string   `yaml:"image"`
	Command []string `yaml:"command"`
	Args    []string `yaml:"args"`
	EnvFrom []struct {
		ConfigMapRef *struct {
			Name string `yaml:"name"`
		} `yaml:"configMapRef"`
		SecretRef *struct {
			Name string `yaml:"name"`
		} `yaml:"secretRef"`
	} `yaml:"envFrom"`
	Env []struct {
		Name      string `yaml:"name"`
		ValueFrom *struct {
			ConfigMapKeyRef *struct {
				Name string `yaml:"name"`
				Key  string `yaml:"key"`
			} `yaml:"configMapKeyRef"`
			SecretKeyRef *struct {
				Name string `yaml:"name"`
				Key  string `yaml:"key"`
			} `yaml:"secretKeyRef"`
		} `yaml:"valueFrom"`
	} `yaml:"env"`
}

func TestListingKitImageAgentDeploymentsAndCanaryAreCapabilityIsolated(t *testing.T) {
	base := filepath.Join("..", "deployments", "kubernetes", "listingkit-workbench")
	for _, name := range []string{"image-agent-temporal-worker-deployment.yaml", "image-agent-temporal-worker-v3-deployment.yaml"} {
		if _, err := os.Stat(filepath.Join(base, "base", name)); !os.IsNotExist(err) {
			t.Fatalf("retired worker %s must be absent: %v", name, err)
		}
	}
	canary := loadImageAgentWorkloadFromMultiDoc(t, filepath.Join(base, "release-authority", "listingkit-release-gate-runners.yaml"), "image-agent-temporal-v3-canary-runner")

	if canary.Kind != "Deployment" || canary.Metadata.Name != "image-agent-temporal-v3-canary-runner" || canary.Spec.Replicas != 0 {
		t.Fatalf("v3 compatibility canary must be a fixed zero-replica Deployment, got kind=%q name=%q replicas=%d", canary.Kind, canary.Metadata.Name, canary.Spec.Replicas)
	}
	if len(canary.Spec.Template.Spec.InitContainers) != 1 {
		t.Fatalf("canary runner must contain one release-gate init container")
	}
	canaryContainer := canary.Spec.Template.Spec.InitContainers[0]
	if !reflect.DeepEqual(canaryContainer.Command, []string{"/app/image-agent-temporal-worker"}) ||
		!reflect.DeepEqual(canaryContainer.Args, []string{"-canary", "-canary-task-queue", "image-agent-manual-v3-canary", "-log-level", "info"}) {
		t.Fatalf("canary must invoke only the side-effect-free v3 compatibility mode, command=%#v args=%#v", canaryContainer.Command, canaryContainer.Args)
	}
	for _, source := range canaryContainer.EnvFrom {
		if source.SecretRef != nil {
			t.Fatalf("compatibility canary must not import Secret %q", source.SecretRef.Name)
		}
	}
	for _, variable := range canaryContainer.Env {
		if variable.ValueFrom != nil && variable.ValueFrom.SecretKeyRef != nil {
			t.Fatalf("compatibility canary must not receive Secret key %q", variable.ValueFrom.SecretKeyRef.Key)
		}
	}
}

func loadImageAgentWorkloadFromMultiDoc(t *testing.T, path, name string) imageAgentWorkloadManifest {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read image-agent workload %s: %v", path, err)
	}
	decoder := yaml.NewDecoder(strings.NewReader(string(content)))
	for {
		var manifest imageAgentWorkloadManifest
		if err := decoder.Decode(&manifest); err != nil {
			t.Fatalf("find workload %s in %s: %v", name, path, err)
		}
		if manifest.Metadata.Name == name {
			return manifest
		}
	}
}

func loadImageAgentWorkloadManifest(t *testing.T, path string) imageAgentWorkloadManifest {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read image-agent workload %s: %v", path, err)
	}
	var manifest imageAgentWorkloadManifest
	if err := yaml.Unmarshal(content, &manifest); err != nil {
		t.Fatalf("parse image-agent workload %s: %v", path, err)
	}
	return manifest
}

func onlyImageAgentContainer(t *testing.T, manifest imageAgentWorkloadManifest) imageAgentWorkloadContainer {
	t.Helper()
	if len(manifest.Spec.Template.Spec.Containers) != 1 {
		t.Fatalf("%s has %d containers, want one", manifest.Metadata.Name, len(manifest.Spec.Template.Spec.Containers))
	}
	return manifest.Spec.Template.Spec.Containers[0]
}

func TestListingKitProductListingAPIImageCarriesV3NewStartsContract(t *testing.T) {
	t.Parallel()

	content, err := os.ReadFile(filepath.Join("..", "deployments", "docker", "Dockerfile.product-listing-api"))
	if err != nil {
		t.Fatal(err)
	}
	const label = `org.opencontainers.image.listingkit.image-agent-routing="image-agent-v3-new-starts-v1"`
	if !strings.Contains(string(content), label) {
		t.Fatalf("product-listing API image must carry immutable v3 new-start routing label %q", label)
	}
}
