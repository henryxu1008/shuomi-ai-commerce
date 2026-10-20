package listingkit_release_authority

import rego.v1

documents := [entry | some entry in input]

release_policies := [entry.contents.releaseAuthority |
  some entry in documents
  entry.contents.releaseAuthority
]

policy := release_policies[0] if count(release_policies) == 1

workflow_documents(owner) := [entry.contents |
  some entry in documents
  endswith(sprintf("%v", [entry.path]), workflow_path(owner))
]

workflow_path("api") := ".github/workflows/listingkit-deploy.yml"
workflow_path("ui") := ".github/workflows/listingkit-ui-deploy.yml"

workflow(owner) := contents if {
	count(workflow_documents(owner)) == 1
	contents := workflow_documents(owner)[0]
}

configmap_documents(name) := [entry.contents |
  some entry in documents
  entry.contents.kind == "ConfigMap"
  entry.contents.metadata.name == name
]

deployment_documents(name) := [entry.contents |
  some entry in documents
  entry.contents.kind == "Deployment"
  entry.contents.metadata.name == name
]

runner_deployments := [entry.contents |
  some entry in documents
  entry.contents.kind == "Deployment"
  endswith(entry.contents.metadata.name, "-runner")
]

runner_names := {runner.metadata.name | some runner in runner_deployments}
expected_runner_names := {name | some name in policy.releaseGateRunners.names}

workflow_steps(owner) := [step |
  some job_name
  job := workflow(owner).jobs[job_name]
  some step in job.steps
]

deny contains "release bundle must contain exactly one machine policy" if count(release_policies) != 1

deny contains msg if {
  owner := ["api", "ui"][_]
  count(workflow_documents(owner)) != 1
  msg := sprintf("release bundle must contain exactly one %v workflow", [owner])
}

# This source bundle retains static safety checks, never publisher identities.
# Paths are fixed here so a policy pointer cannot hide a missing required input.
assembly_path := "deployments/kubernetes/listingkit-workbench/release-authority/kustomization.yaml"
assembly_documents := [entry.contents |
  some entry in documents
  endswith(sprintf("%v", [entry.path]), assembly_path)
]

deny contains "machine policy must be an object" if not is_object(policy)

deny contains msg if {
  field := ["repository", "namespace", "tools", "assembly", "uiAuthorization", "releaseGateRunners", "documentation"][_]
  not field in object.keys(policy)
  msg := sprintf("machine policy is missing required field %v", [field])
}

deny contains msg if {
  field := ["oidc", "rbac", "protectedResources"][_]
  field in object.keys(policy)
  msg := sprintf("retired machine policy field %v must be absent", [field])
}

deny contains msg if {
  some entry in documents
  entry.contents.kind in {"Role", "RoleBinding", "AuthenticationConfiguration"}
  msg := sprintf("release bundle must not contain retired authority object %v", [entry.contents.kind])
}

deny contains "machine policy assembly path drifted" if not policy.assembly == {"kustomization": assembly_path}

deny contains "release bundle must contain exactly one governed assembly" if count(assembly_documents) != 1

deny contains "release assembly must contain only the current canary manifest" if {
  some assembly in assembly_documents
  assembly != {
    "apiVersion": "kustomize.config.k8s.io/v1beta1",
    "kind": "Kustomization",
    "namespace": "task-processor",
    "resources": ["listingkit-release-gate-runners.yaml"],
  }
}

deny contains "machine policy must pin Conftest and OPA" if {
  not policy.tools.conftestImage == "openpolicyagent/conftest@sha256:5fd81e332d7e4bc01daf3ef35371800a9a9720a30c0c37a78de0c5fbe4b6d622"
}

deny contains "release-gate runner policy drifted" if {
  not policy.releaseGateRunners == {
    "manifest": "deployments/kubernetes/listingkit-workbench/release-authority/listingkit-release-gate-runners.yaml",
    "holdImage": "registry.k8s.io/pause@sha256:ee6521f290b2168b6e0935a181d4cff9be1ac3f505666ef0e3c98fae8199917a",
    "names": ["image-agent-temporal-v3-canary-runner"],
  }
}

deny contains "release-gate runner inventory drifted" if runner_names != expected_runner_names

deny contains msg if {
  some runner in runner_deployments
  runner.spec.replicas != 0
  msg := sprintf("release-gate runner %v must start at zero replicas", [runner.metadata.name])
}

deny contains msg if {
  some runner in runner_deployments
  count(runner.spec.template.spec.initContainers) != 1
  msg := sprintf("release-gate runner %v must have exactly one init container", [runner.metadata.name])
}

deny contains msg if {
  some runner in runner_deployments
  some init in runner.spec.template.spec.initContainers
  init.name == "release-gate"
  "restartPolicy" in object.keys(init)
  msg := sprintf("release-gate runner %v init container must not set restartPolicy", [runner.metadata.name])
}

deny contains msg if {
  some runner in runner_deployments
  count(runner.spec.template.spec.containers) != 1
  msg := sprintf("release-gate runner %v must have exactly one hold container", [runner.metadata.name])
}

deny contains msg if {
  some runner in runner_deployments
  runner.spec.template.spec.containers[0].image != policy.releaseGateRunners.holdImage
  msg := sprintf("release-gate runner %v hold image drifted", [runner.metadata.name])
}

deny contains "machine policy repository or namespace drifted" if {
  count(release_policies) == 1
  not policy.repository == "qq550723504/task-processor"
}

deny contains "machine policy repository or namespace drifted" if {
  count(release_policies) == 1
  not policy.namespace == "task-processor"
}

deny contains "machine policy must pin Conftest and OPA" if {
  count(release_policies) == 1
  not policy.tools.conftestVersion == "0.68.2"
}

deny contains "UI authorization ownership policy drifted" if {
  not policy.uiAuthorization == {
    "owner": "ui",
    "sharedConfigMap": "listingkit-workbench-config",
    "configMap": "listingkit-ui-auth-config",
    "deployment": "listingkit-ui",
    "container": "listingkit-ui",
    "scopesKey": "ZITADEL_SCOPES",
  }
}

deny contains "UI authorization ConfigMap inventory drifted" if {
  count(configmap_documents(policy.uiAuthorization.sharedConfigMap)) != 1
}

deny contains "UI authorization ConfigMap inventory drifted" if {
  count(configmap_documents(policy.uiAuthorization.configMap)) != 1
}

deny contains "UI authorization scopes leaked into shared runtime configuration" if {
  shared := configmap_documents(policy.uiAuthorization.sharedConfigMap)[0]
  policy.uiAuthorization.scopesKey in object.keys(shared.data)
}

deny contains "UI authorization scopes are missing from the UI-owned ConfigMap" if {
  ui := configmap_documents(policy.uiAuthorization.configMap)[0]
  object.get(object.get(ui, "data", {}), policy.uiAuthorization.scopesKey, "") == ""
}

deny contains "UI deployment authorization ConfigMap ownership drifted" if {
  count(deployment_documents(policy.uiAuthorization.deployment)) != 1
}

deny contains "UI deployment authorization ConfigMap ownership drifted" if {
  deployment := deployment_documents(policy.uiAuthorization.deployment)[0]
  containers := [container |
    some container in deployment.spec.template.spec.containers
    container.name == policy.uiAuthorization.container
  ]
  count(containers) != 1
}

deny contains "UI deployment authorization ConfigMap ownership drifted" if {
  deployment := deployment_documents(policy.uiAuthorization.deployment)[0]
  container := [candidate |
    some candidate in deployment.spec.template.spec.containers
    candidate.name == policy.uiAuthorization.container
  ][0]
  configmap_refs := {source.configMapRef.name |
    some source in container.envFrom
    source.configMapRef
  }
  configmap_refs != {policy.uiAuthorization.sharedConfigMap, policy.uiAuthorization.configMap}
}

deny contains "machine policy must pin Conftest and OPA" if {
  count(release_policies) == 1
  not policy.tools.opaVersion == "1.15.2"
}

deny contains msg if {
  owner := ["api", "ui"][_]
  contains(json.marshal(workflow(owner)), "KUBE_CONFIG")
  msg := sprintf("%v workflow retains a long-lived kubeconfig credential", [owner])
}

# A tag is human-readable metadata, never executable release authority. All
# third-party actions on either production workflow path must be immutable commits.
deny contains msg if {
  owner := ["api", "ui"][_]
  some step in workflow_steps(owner)
  action := step.uses
  not regex.match("^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+@[0-9a-f]{40}$", action)
  msg := sprintf("%v release action is not pinned to a full commit SHA", [owner])
}

# These two source entrypoints are retired under the confirmed internal-trial
# contract. Compare the complete parsed document; no marker, skip or bypass.
retired_workflow_name("api") := "ListingKit API Deploy"
retired_workflow_name("ui") := "ListingKit UI Deploy"
retired_owner_name("api") := "API"
retired_owner_name("ui") := "UI"

expected_retired_workflow(owner) := {
  "name": retired_workflow_name(owner),
  "on": {"workflow_dispatch": {}},
  "permissions": {},
  "jobs": {"retired": {
    "runs-on": "ubuntu-latest",
    "steps": [{"run": sprintf("echo \"::error::Legacy ListingKit %v release and rollback are retired; current-application production release is NOT_READY.\"\nexit 1\n", [retired_owner_name(owner)])}],
  }},
}

deny contains msg if {
  owner := ["api", "ui"][_]
  workflow(owner) != expected_retired_workflow(owner)
  msg := sprintf("%v retired release entry must be exactly the permission-free fixed failure", [owner])
}
