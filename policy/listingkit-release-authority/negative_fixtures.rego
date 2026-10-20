package listingkit_release_fixture

import rego.v1

deny contains "long-lived kubeconfig fallback" if input.workflow.credentials[_] == "KUBE_CONFIG"
deny contains "reusable workflow bypass" if input.workflow.uses
deny contains "composite action bypass" if startswith(input.workflow.stepUses, "./")
deny contains "release-gate Pod query is forbidden" if contains(input.releaseGateCommand, "get pods")
deny contains "machine policy drift" if input.policy.repository != "qq550723504/task-processor"
deny contains "mutable third-party action ref" if not regex.match("^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+@[0-9a-f]{40}$", input.workflow.actionUses)
