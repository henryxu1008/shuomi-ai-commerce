package record

import (
	"errors"
	"strings"
	"testing"
)

func TestActorValidation(t *testing.T) {
	valid := Actor{OrganizationID: "tenant-1", UserID: "user-1", Roles: []string{"listingkit_operator"}}
	if err := ValidateActor(valid); err != nil {
		t.Fatalf("ValidateActor(valid) error = %v", err)
	}

	tests := []struct {
		name  string
		actor Actor
	}{
		{name: "empty tenant", actor: Actor{UserID: valid.UserID, Roles: valid.Roles}},
		{name: "blank tenant", actor: Actor{OrganizationID: " ", UserID: valid.UserID, Roles: valid.Roles}},
		{name: "trimmed tenant", actor: Actor{OrganizationID: " tenant-1", UserID: valid.UserID, Roles: valid.Roles}},
		{name: "long tenant", actor: Actor{OrganizationID: strings.Repeat("t", maxActorOrganizationBytes+1), UserID: valid.UserID, Roles: valid.Roles}},
		{name: "empty user", actor: Actor{OrganizationID: valid.OrganizationID, Roles: valid.Roles}},
		{name: "blank user", actor: Actor{OrganizationID: valid.OrganizationID, UserID: " ", Roles: valid.Roles}},
		{name: "trimmed user", actor: Actor{OrganizationID: valid.OrganizationID, UserID: "user-1 ", Roles: valid.Roles}},
		{name: "long user", actor: Actor{OrganizationID: valid.OrganizationID, UserID: strings.Repeat("u", maxActorUserBytes+1), Roles: valid.Roles}},
		{name: "empty roles", actor: Actor{OrganizationID: valid.OrganizationID, UserID: valid.UserID}},
		{name: "blank role", actor: Actor{OrganizationID: valid.OrganizationID, UserID: valid.UserID, Roles: []string{" "}}},
		{name: "trimmed role", actor: Actor{OrganizationID: valid.OrganizationID, UserID: valid.UserID, Roles: []string{" role"}}},
		{name: "long role", actor: Actor{OrganizationID: valid.OrganizationID, UserID: valid.UserID, Roles: []string{strings.Repeat("r", maxActorRoleBytes+1)}}},
		{name: "too many roles", actor: Actor{OrganizationID: valid.OrganizationID, UserID: valid.UserID, Roles: make([]string, maxActorRoles+1)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateActor(tt.actor); !errors.Is(err, ErrInvalid) {
				t.Fatalf("ValidateActor() error = %v, want %v", err, ErrInvalid)
			}
		})
	}
}

func TestOperationIDValidation(t *testing.T) {
	for _, taskID := range []string{"", " ", " task-1", "task-1 ", strings.Repeat("t", maxOperationIDBytes+1)} {
		if err := ValidateOperationID(taskID); !errors.Is(err, ErrInvalid) {
			t.Fatalf("ValidateOperationID(%q) error = %v, want %v", taskID, err, ErrInvalid)
		}
	}
	if err := ValidateOperationID(strings.Repeat("t", maxOperationIDBytes)); err != nil {
		t.Fatalf("ValidateOperationID(max) error = %v", err)
	}
}

func TestActorValidationAcceptsExactByteAndRoleLimits(t *testing.T) {
	actor := Actor{OrganizationID: strings.Repeat("o", maxActorOrganizationBytes), UserID: strings.Repeat("u", maxActorUserBytes), Roles: make([]string, maxActorRoles)}
	for i := range actor.Roles {
		actor.Roles[i] = strings.Repeat("r", maxActorRoleBytes)
	}
	if err := ValidateActor(actor); err != nil {
		t.Fatalf("exact limits must remain valid: %v", err)
	}
	actor.OrganizationID = strings.Repeat("界", 22)
	if !errors.Is(ValidateActor(actor), ErrInvalid) {
		t.Fatal("organization limit is bytes, not runes")
	}
}
