package channelcredential

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCreateRegistrationUsesLiveScopeAndStableIdentity(t *testing.T) {
	input := RegistrationInput{Template: CompatibleMiniTemplate, Name: "采购渠道", BaseURL: "https://MODELS.example.com:443/v1/", APIKey: "fixture-key"}
	first, err := NewRegistrationCommand("org-a", "actor-a", "request-a", input)
	if err != nil {
		t.Fatal(err)
	}
	repeated, err := NewRegistrationCommand("org-a", "actor-a", "request-a", input)
	if err != nil || repeated != first || !IsRegisteredConnection(first.ConnectionID) {
		t.Fatalf("create identity was not stable: %v", err)
	}
	for _, scope := range [][2]string{{"org-b", "actor-a"}, {"org-a", "actor-b"}} {
		other, err := NewRegistrationCommand(scope[0], scope[1], "request-a", input)
		if err != nil || other.ConnectionID == first.ConnectionID {
			t.Fatalf("create identity did not bind scope: %v", err)
		}
	}
	if first.BaseURL != "https://models.example.com/v1" || first.Channel != "openai-compatible" {
		t.Fatal("connection did not use the canonical compatible template")
	}
	raw, _ := json.Marshal(first)
	if strings.Contains(string(raw), input.APIKey) {
		t.Fatal("serialized command disclosed Key")
	}
}

func TestRegistrationRejectsNonPublicOrNonCanonicalDestinations(t *testing.T) {
	for _, address := range []string{
		"http://models.example.com/v1", "https://localhost/v1", "https://127.0.0.1/v1",
		"https://[::ffff:8.8.8.8]/v1", "https://169.254.169.254/v1",
		"https://models.example.com:8443/v1", "https://models.example.com/v1?",
		"https://models.example.com/v1#fragment", "https://models.example.com/v1/%2e%2e",
		"https://models.example.com/v1/chat/completions", "https://key@models.example.com/v1",
	} {
		t.Run(address, func(t *testing.T) {
			input := RegistrationInput{Template: CompatibleMiniTemplate, Name: "渠道", BaseURL: address, APIKey: "fixture"}
			if _, err := NewRegistrationCommand("org", "actor", "request", input); err == nil {
				t.Fatal("unexpectedly accepted destination")
			}
		})
	}
	input := RegistrationInput{Template: OfficialMiniTemplate, Name: "官方", BaseURL: "https://models.example.com/v1", APIKey: "fixture"}
	if _, err := NewRegistrationCommand("org", "actor", "request", input); err == nil {
		t.Fatal("an official registration must not accept a submitted address")
	}
}

func TestGRSaiImageRegistrationOwnsFixedProviderEndpoint(t *testing.T) {
	input := RegistrationInput{Template: GRSaiImageTemplate, Name: "商品图片", APIKey: "fixture"}
	registered, err := NewRegistrationCommand("enterprise", "editor", "request", input)
	if err != nil || registered.Validate() != nil || registered.Provider != "grsai" || registered.BaseURL != "https://grsaiapi.com" || registered.Channel != GRSaiImageChannel {
		t.Fatal("GRSai image registration must use the fixed enterprise provider")
	}
	if _, err := NewRegistrationCommand(PlatformOwner, "editor", "request", input); err == nil {
		t.Fatal("GRSai image registration must not admit a platform credential")
	}
	domestic := input
	domestic.BaseURL = "https://grsai.dakka.com.cn"
	cn, err := NewRegistrationCommand("enterprise", "editor", "domestic-request", domestic)
	if err != nil || cn.Validate() != nil || cn.Region != "cn" || cn.BaseURL != domestic.BaseURL || cn.ConnectionID == registered.ConnectionID {
		t.Fatalf("GRSai domestic must use a new fixed connection: %v", err)
	}
	cn.Region = "global"
	if cn.Validate() == nil {
		t.Fatal("mismatching a region with its supplier address must fail")
	}
	domestic.BaseURL += "/"
	if _, err := NewRegistrationCommand("enterprise", "editor", "invalid-domestic-request", domestic); err == nil {
		t.Fatal("regional template must reject all other addresses")
	}
	input.BaseURL = "https://grsaiapi.com"
	if _, err := NewRegistrationCommand("enterprise", "editor", "request", input); err == nil {
		t.Fatal("GRSai image registration must not take a caller-selected URL")
	}
}
