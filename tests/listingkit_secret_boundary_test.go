package tests

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"gopkg.in/yaml.v3"
)

const listingKitSharedSecret = "listingkit-workbench-secret"

type workloadManifest struct {
	Kind     string `yaml:"kind"`
	Metadata struct {
		Name         string `yaml:"name"`
		GenerateName string `yaml:"generateName"`
	} `yaml:"metadata"`
	Spec struct {
		Template podTemplate `yaml:"template"`
	} `yaml:"spec"`
}

type podTemplate struct {
	Spec struct {
		Containers []workloadContainer `yaml:"containers"`
	} `yaml:"spec"`
}

type workloadContainer struct {
	Name         string   `yaml:"name"`
	Args         []string `yaml:"args"`
	VolumeMounts []struct {
		Name      string `yaml:"name"`
		MountPath string `yaml:"mountPath"`
		SubPath   string `yaml:"subPath"`
		ReadOnly  bool   `yaml:"readOnly"`
	} `yaml:"volumeMounts"`
	EnvFrom []struct {
		SecretRef *struct {
			Name     string `yaml:"name"`
			Optional *bool  `yaml:"optional"`
		} `yaml:"secretRef"`
	} `yaml:"envFrom"`
	Env []struct {
		Name      string `yaml:"name"`
		ValueFrom *struct {
			SecretKeyRef *struct {
				Name     string `yaml:"name"`
				Key      string `yaml:"key"`
				Optional *bool  `yaml:"optional"`
			} `yaml:"secretKeyRef"`
		} `yaml:"valueFrom"`
	} `yaml:"env"`
}

func TestListingKitWorkloadsUseLeastPrivilegeSharedSecretKeys(t *testing.T) {
	want := map[string]map[string]string{
		"base/listingkit-ui-deployment.yaml": {
			"AUTH_SECRET":           "AUTH_SECRET",
			"ZITADEL_ISSUER_URL":    "ZITADEL_ISSUER_URL",
			"ZITADEL_CLIENT_ID":     "ZITADEL_CLIENT_ID",
			"ZITADEL_CLIENT_SECRET": "ZITADEL_CLIENT_SECRET",
			"TASK_PROCESSOR_LISTINGKIT_ZITADEL_ALLOWED_TENANT_IDS": "TASK_PROCESSOR_LISTINGKIT_ZITADEL_ALLOWED_TENANT_IDS",
			"TASK_PROCESSOR_LISTINGKIT_ZITADEL_ALLOWED_USER_IDS":   "TASK_PROCESSOR_LISTINGKIT_ZITADEL_ALLOWED_USER_IDS",
			"TASK_PROCESSOR_LISTINGKIT_ZITADEL_ALLOWED_ROLES":      "TASK_PROCESSOR_LISTINGKIT_ZITADEL_ALLOWED_ROLES",
			"LISTINGKIT_DEMO_WEBHOOK_URL":                          "LISTINGKIT_DEMO_WEBHOOK_URL",
		},
		"base/shein-login-worker-deployment.yaml": {},
		"base/imgproxy-deployment.yaml": {
			"IMGPROXY_KEY":          "IMGPROXY_KEY",
			"IMGPROXY_SALT":         "IMGPROXY_SALT",
			"AWS_ACCESS_KEY_ID":     "TASK_PROCESSOR_LISTINGKIT_IMAGE_UPLOAD_S3_ACCESS_KEY_ID",
			"AWS_SECRET_ACCESS_KEY": "TASK_PROCESSOR_LISTINGKIT_IMAGE_UPLOAD_S3_SECRET_ACCESS_KEY",
		},
	}

	for relativePath, expected := range want {
		t.Run(relativePath, func(t *testing.T) {
			container := loadOnlyContainer(t, relativePath)
			for _, source := range container.EnvFrom {
				if source.SecretRef != nil {
					t.Fatalf("%s must not import Secret %s with envFrom", relativePath, source.SecretRef.Name)
				}
			}

			actual := make(map[string]string)
			forbidden := map[string]bool{
				"TASK_PROCESSOR_LISTINGKIT_ZITADEL_TENANT_DIRECTORY_TOKEN":  true,
				"TASK_PROCESSOR_LISTINGKIT_ZITADEL_MEMBER_INVITATION_TOKEN": true,
			}
			for _, variable := range container.Env {
				if variable.ValueFrom == nil || variable.ValueFrom.SecretKeyRef == nil {
					continue
				}
				ref := variable.ValueFrom.SecretKeyRef
				if forbidden[ref.Key] {
					t.Fatalf("%s must not receive forbidden key %s", relativePath, ref.Key)
				}
				if ref.Name != listingKitSharedSecret {
					t.Fatalf("%s references unexpected Secret %s", relativePath, ref.Name)
				}
				actual[variable.Name] = ref.Key
			}
			if !reflect.DeepEqual(actual, expected) {
				t.Fatalf("%s shared Secret allowlist = %#v, want %#v", relativePath, actual, expected)
			}
		})
	}
}

func TestListingKitSecretExamplesDefineCanonicalUIAllowlists(t *testing.T) {
	for _, relativePath := range []string{
		"listingkit-workbench/base/secret.example.yaml",
		"zitadel/local/listingkit-workbench-zitadel-secret.example.yaml",
	} {
		t.Run(relativePath, func(t *testing.T) {
			path := filepath.Join("..", "deployments", "kubernetes", filepath.FromSlash(relativePath))
			contents, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", relativePath, err)
			}
			var secret struct {
				StringData map[string]string `yaml:"stringData"`
			}
			if err := yaml.Unmarshal(contents, &secret); err != nil {
				t.Fatalf("parse %s: %v", relativePath, err)
			}
			for _, key := range []string{
				"TASK_PROCESSOR_LISTINGKIT_ZITADEL_ALLOWED_TENANT_IDS",
				"TASK_PROCESSOR_LISTINGKIT_ZITADEL_ALLOWED_USER_IDS",
				"TASK_PROCESSOR_LISTINGKIT_ZITADEL_ALLOWED_ROLES",
			} {
				if _, ok := secret.StringData[key]; !ok {
					t.Errorf("%s must define required canonical UI allowlist key %s", relativePath, key)
				}
			}
			for _, key := range []string{
				"LISTINGKIT_ZITADEL_ALLOWED_TENANT_IDS",
				"LISTINGKIT_ZITADEL_ALLOWED_USER_IDS",
				"LISTINGKIT_ZITADEL_ALLOWED_ROLES",
			} {
				if _, ok := secret.StringData[key]; ok {
					t.Errorf("%s must not define deprecated UI allowlist key %s", relativePath, key)
				}
			}
		})
	}
}

func loadOnlyContainer(t *testing.T, relativePath string) workloadContainer {
	t.Helper()
	path := filepath.Join("..", "deployments", "kubernetes", "listingkit-workbench", filepath.FromSlash(relativePath))
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var manifest workloadManifest
	if err := yaml.Unmarshal(contents, &manifest); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	if len(manifest.Spec.Template.Spec.Containers) != 1 {
		t.Fatalf("%s has %d containers, want exactly one", path, len(manifest.Spec.Template.Spec.Containers))
	}
	return manifest.Spec.Template.Spec.Containers[0]
}

func TestSheinWorkerUsesOnlyFreshPrivateFileInputs(t *testing.T) {
	c := loadOnlyContainer(t, "base/shein-login-worker-deployment.yaml")
	if len(c.Env) != 0 || len(c.EnvFrom) != 0 {
		t.Fatal("worker must not import numeric environment or shared ConfigMap")
	}
	if len(c.Args) < 2 || c.Args[0] != "-config" || c.Args[1] != "/run/shein-login-worker/current.json" {
		t.Fatal("explicit current config missing")
	}
	required := map[string]bool{"current.json": false, "client-secret": false, "service-token": false, "login.json": false}
	for _, m := range c.VolumeMounts {
		if m.Name != "worker-current" {
			continue
		}
		if _, ok := required[m.SubPath]; !ok || !m.ReadOnly || m.MountPath != "/run/shein-login-worker/"+m.SubPath {
			t.Fatal("private regular-file mount mismatch")
		}
		required[m.SubPath] = true
	}
	for key, present := range required {
		if !present {
			t.Fatal("missing private file", key)
		}
	}
	path := filepath.Join("..", "deployments", "kubernetes", "listingkit-workbench", "base", "shein-login-worker-deployment.yaml")
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Spec struct {
			Template struct {
				Spec struct {
					Volumes []struct {
						Name   string
						Secret *struct {
							SecretName  string `yaml:"secretName"`
							DefaultMode int    `yaml:"defaultMode"`
							Optional    bool
						}
					}
				}
			}
		}
	}
	if err = yaml.Unmarshal(contents, &manifest); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, v := range manifest.Spec.Template.Spec.Volumes {
		if v.Name == "worker-current" {
			found = true
			if v.Secret == nil || v.Secret.SecretName != "shein-login-worker-current" || v.Secret.DefaultMode != 0400 || v.Secret.Optional {
				t.Fatal("private current Secret boundary mismatch")
			}
		}
	}
	if !found {
		t.Fatal("current private Secret missing")
	}
}
