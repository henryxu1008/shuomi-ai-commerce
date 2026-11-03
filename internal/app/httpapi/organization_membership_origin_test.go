package httpapi

import "testing"

func TestMembershipAssemblyAdmitsFixedHTTPSIdentityOrigin(t *testing.T) {
	for _, origin := range []string{"https://auth-trial.shuomiai.com", "https://localhost:18443", "http://127.0.0.1:8800"} {
		if err := validateMembershipProviderOrigin(origin, origin); err != nil {
			t.Fatalf("admitted membership origin %s rejected: %v", origin, err)
		}
	}
	for _, candidate := range []string{"https://other.example.com", "http://auth-trial.shuomiai.com", "https://auth-trial.shuomiai.com/", "https://auth-trial.shuomiai.com:443", "https://user@auth-trial.shuomiai.com", "https://auth-trial.shuomiai.com?query=1", "https://auth-trial.shuomiai.com#fragment", "https://localhost:18443"} {
		if validateMembershipProviderOrigin(candidate, "https://auth-trial.shuomiai.com") == nil {
			t.Fatalf("unadmitted membership origin %s accepted", candidate)
		}
	}
}
