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

role(owner) := contents if {
  identity := policy.oidc.identities[owner]
  some entry in documents
  entry.contents.kind == "Role"
  entry.contents.metadata.name == identity.role
  contents := entry.contents
}

role_binding(owner) := contents if {
  identity := policy.oidc.identities[owner]
  some entry in documents
  entry.contents.kind == "RoleBinding"
  entry.contents.roleRef.name == identity.role
  contents := entry.contents
}

authentication_config := contents if {
  some entry in documents
  entry.contents.kind == "AuthenticationConfiguration"
  contents := entry.contents
}

role_documents(owner) := [entry.contents |
  identity := policy.oidc.identities[owner]
  some entry in documents
  entry.contents.kind == "Role"
  entry.contents.metadata.name == identity.role
]

role_binding_documents(owner) := [entry.contents |
  identity := policy.oidc.identities[owner]
  some entry in documents
  entry.contents.kind == "RoleBinding"
  entry.contents.roleRef.name == identity.role
]

authentication_configs := [entry.contents |
  some entry in documents
  entry.contents.kind == "AuthenticationConfiguration"
]

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

rule_signatures(owner) := {signature |
  some rule in role(owner).rules
  some resource in rule.resources
  signature := sprintf("%v|%v|%v|%v", [sort(rule.apiGroups), resource, sort(rule.resourceNames), sort(rule.verbs)])
}

expected_rule_signatures(owner) := {signature |
  some protected in policy.protectedResources
  protected.owner == owner
  signature := sprintf("%v|%v|%v|%v", [sort(protected.apiGroups), protected.resource, sort(protected.resourceNames), sort(protected.verbs)])
}

deny contains "release bundle must contain exactly one machine policy" if count(release_policies) != 1

deny contains msg if {
  owner := ["api", "ui"][_]
  count(workflow_documents(owner)) != 1
  msg := sprintf("release bundle must contain exactly one %v workflow", [owner])
}

deny contains msg if {
  owner := ["api", "ui"][_]
  count(role_documents(owner)) != 1
  msg := sprintf("release bundle must contain exactly one %v Role", [owner])
}

deny contains msg if {
  owner := ["api", "ui"][_]
  count(role_binding_documents(owner)) != 1
  msg := sprintf("release bundle must contain exactly one %v RoleBinding", [owner])
}

deny contains "release bundle must contain exactly one AuthenticationConfiguration" if count(authentication_configs) != 1

deny contains msg if {
  owner := ["api", "ui"][_]
  identity := policy.oidc.identities[owner]
  identity.workflowRef != sprintf("%v/%v@refs/heads/main", [policy.repository, workflow_path(owner)])
  msg := sprintf("%v trusted workflow ref drifted", [owner])
}

deny contains "release-gate runner inventory drifted" if runner_names != expected_runner_names

deny contains "API release Role must not grant Pod permissions" if {
  some rule in role("api").rules
  startswith(rule.resources[_], "pods")
}

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
  policy.repository != "qq550723504/task-processor"
}

deny contains "machine policy repository or namespace drifted" if {
  count(release_policies) == 1
  policy.namespace != "task-processor"
}

deny contains "machine policy must pin Conftest and OPA" if {
  count(release_policies) == 1
  policy.tools.conftestVersion != "0.68.2"
}

deny contains "UI authorization ownership policy drifted" if {
  policy.uiAuthorization != {
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
  object.get(ui.data, policy.uiAuthorization.scopesKey, "") == ""
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
  policy.tools.opaVersion != "1.15.2"
}

deny contains msg if {
  owner := ["api", "ui"][_]
  contains(json.marshal(workflow(owner)), "KUBE_CONFIG")
  msg := sprintf("%v workflow retains a long-lived kubeconfig credential", [owner])
}

deny contains msg if {
  owner := ["api", "ui"][_]
  rule_signatures(owner) != expected_rule_signatures(owner)
  msg := sprintf("%v Kubernetes Role differs from the protected-resource policy", [owner])
}

deny contains msg if {
  owner := ["api", "ui"][_]
  identity := policy.oidc.identities[owner]
  binding := role_binding(owner)
  count(binding.subjects) != 1
  msg := sprintf("%v RoleBinding subject drifted", [owner])
}

deny contains msg if {
  owner := ["api", "ui"][_]
  identity := policy.oidc.identities[owner]
  binding := role_binding(owner)
  binding.subjects[0].kind != "User"
  msg := sprintf("%v RoleBinding subject drifted", [owner])
}

deny contains msg if {
  owner := ["api", "ui"][_]
  identity := policy.oidc.identities[owner]
  binding := role_binding(owner)
  binding.subjects[0].name != sprintf("%v%v", [policy.oidc.usernamePrefix, identity.subject])
  msg := sprintf("%v RoleBinding subject drifted", [owner])
}

deny contains "API and UI release identities must be distinct" if {
  policy.oidc.identities.api.environment == policy.oidc.identities.ui.environment
}

deny contains "API and UI release identities must be distinct" if {
  policy.oidc.identities.api.audience == policy.oidc.identities.ui.audience
}

deny contains "API and UI release identities must be distinct" if {
  policy.oidc.identities.api.subject == policy.oidc.identities.ui.subject
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

# Owner overlap would allow the UI identity to alter API or worker state.
deny contains "API and UI protected resources overlap" if {
  api := policy.protectedResources[_]
  ui := policy.protectedResources[_]
  api.owner == "api"
  ui.owner == "ui"
  api.apiGroups == ui.apiGroups
  api.resource == ui.resource
  name := api.resourceNames[_]
  name == ui.resourceNames[_]
}

deny contains "Kubernetes authentication issuer or audience drifted" if {
  auth := authentication_config.jwt[0]
  auth.issuer.url != policy.oidc.issuer
}

deny contains "Kubernetes authentication issuer or audience drifted" if {
  auth := authentication_config.jwt[0]
  {x | some x in auth.issuer.audiences} != {policy.oidc.identities.api.audience, policy.oidc.identities.ui.audience}
}

deny contains "Kubernetes authentication claim rules do not bind both exact release identities" if {
  encoded := json.marshal(authentication_config)
  not contains(encoded, policy.oidc.identities.api.subject)
}

deny contains "Kubernetes authentication claim rules do not bind both exact release identities" if {
  encoded := json.marshal(authentication_config)
  not contains(encoded, policy.oidc.identities.ui.subject)
}

deny contains "Kubernetes authentication claim rules do not bind both exact release identities" if {
  encoded := json.marshal(authentication_config)
  not contains(encoded, policy.oidc.identities.api.workflowRef)
}

deny contains "Kubernetes authentication claim rules do not bind both exact release identities" if {
  encoded := json.marshal(authentication_config)
  not contains(encoded, policy.oidc.identities.ui.workflowRef)
}

deny contains "Kubernetes authentication claim rules use workflow-ref prefix authorization" if {
  contains(json.marshal(authentication_config), "workflow_ref.startsWith")
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
