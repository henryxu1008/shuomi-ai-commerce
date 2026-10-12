package agentconfig

import (
	"strings"
	"testing"

	"task-processor/internal/agent"
	"task-processor/internal/commercetool"
)

func TestExecutionModeCommandsRejectUnboundIdentity(t *testing.T) {
	public := ModeSubject{Kind: RegisteredAgentSubject, ID: "product.title.agent"}
	private := ModeSubject{Kind: PrivateDeliverySubject, ID: "1dc92db7-6e38-4d89-9b71-9b12432f3678"}
	if !public.Valid() || !private.Valid() {
		t.Fatal("current subject rejected")
	}
	for _, s := range []ModeSubject{{}, {Kind: PrivateDeliverySubject, ID: public.ID}, {Kind: ModeSubjectKind("organization"), ID: "org"}, {Kind: RegisteredAgentSubject, ID: " other "}} {
		if s.Valid() {
			t.Fatalf("invalid subject admitted: %+v", s)
		}
	}
	resolved := ResolvedModeSubject{Subject: public, Definition: commercetool.AgentRef{ID: public.ID, Version: "v1.0.0"}}
	command := ModeCommand{Identity: agent.ExecutionIdentity{OrganizationID: "org", ActorID: "actor", MemberID: "member"}, Subject: resolved, Key: "e1b16b08-46e7-4a6d-9059-2588e2822e69", Mode: agent.ModeReadOnly, ExpectedRevision: 0}
	if command.Validate() != nil {
		t.Fatal("initial explicit mode command rejected")
	}
	bad := command
	bad.Identity.MemberID = ""
	if bad.Validate() == nil {
		t.Fatal("missing member admitted")
	}
	bad = command
	bad.Subject.Definition.ID = "product.image.agent"
	if bad.Validate() == nil {
		t.Fatal("public subject definition substitution admitted")
	}
	bad = command
	bad.Mode = "fully_unrestricted"
	if bad.Validate() == nil {
		t.Fatal("unknown mode admitted")
	}
	start := ModeStartCommand{Identity: command.Identity, Subject: resolved, Binding: ModeBinding{ContextKind: "product_batch", ContextID: "batch", RequestKey: command.Key, IntentDigest: strings.Repeat("ab", 32)}}
	if start.Validate() != nil {
		t.Fatal("bound start rejected")
	}
	start.Binding.IntentDigest = strings.ToUpper(start.Binding.IntentDigest)
	if start.Validate() == nil {
		t.Fatal("noncanonical digest admitted")
	}
}

func TestExecutionModeRevisionIsExactDecimal(t *testing.T) {
	for _, input := range []string{"", "-1", "00", "01", "+1", "1.0", " 1", "9223372036854775808"} {
		if _, err := ParseModeRevision(input); err == nil {
			t.Fatalf("invalid revision admitted %q", input)
		}
	}
	for _, input := range []string{"0", "1", "9223372036854775807"} {
		if _, err := ParseModeRevision(input); err != nil {
			t.Fatalf("canonical revision rejected %q: %v", input, err)
		}
	}
}

func TestExecutionModePrivateSubjectPreservesDeliveryVersion(t *testing.T) {
	resolved := ResolvedModeSubject{Subject: ModeSubject{Kind: PrivateDeliverySubject, ID: "1dc92db7-6e38-4d89-9b71-9b12432f3678"}, Definition: commercetool.AgentRef{ID: "product.quality.check", Version: "2.0.0"}}
	if !resolved.Valid() { t.Fatal("actual private delivery version must not be normalized into a public catalog version") }
}
