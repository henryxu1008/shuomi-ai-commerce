package agentconfigpersistence

import (
	"github.com/henryxu1008/shuomi-ai-commerce/internal/agent"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/agentconfig"
	reviewproposal "github.com/henryxu1008/shuomi-ai-commerce/internal/product/review/proposal"
	"strings"
	"testing"
)

func TestScriptConfigurationCannotFreezeTextOnlyOrUnboundExecution(t *testing.T) {
	request := agent.Request{PolicyVersion: reviewproposal.ContentPolicyVersion, PromptVersion: agentconfig.ContentPromptVersion, Binding: agent.Binding{TargetPlatform: "product"}, ContentSelection: reviewproposal.ContentSelection{Title: true, TargetLanguage: "en"}, SkillVersionRef: agent.SkillVersionRef{Kind: "skill-version-v1", ID: "d68f289c-2e45-4d0a-9a13-d53db4425739", OwnerKind: "USER", OwnerID: "actor", Version: 1, Digest: strings.Repeat("a", 64), PolicyVersion: "skill-script-method-v1"}}
	if validTextConfigurationRequest(agentconfig.ContentAgentID, request) {
		t.Fatal("unbound script frozen as ordinary text")
	}
	request.ScriptDescriptor = agent.ScriptDescriptor{Profile: "python-title-json-v1", Revision: "1", NetworkRevision: "1", ToolVersion: "v1.0.0", ImageDigest: "sha256:" + strings.Repeat("a", 64), ExecdDigest: "sha256:" + strings.Repeat("b", 64)}
	if !validTextConfigurationRequest(agentconfig.ContentAgentID, request) {
		t.Fatal("bounded descriptor not accepted")
	}
	request.SkillVersionRef.PolicyVersion = "skill-method-v1"
	if validTextConfigurationRequest(agentconfig.ContentAgentID, request) {
		t.Fatal("text selected with hidden script descriptor")
	}
}
