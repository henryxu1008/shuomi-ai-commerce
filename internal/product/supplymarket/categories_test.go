package supplymarket

import (
	"errors"
	"testing"
)

func TestPlatformApprovalAndPublicationRequireMarketCategory(t *testing.T) {
	for _, action := range []string{"approve", "publish"} {
		t.Run(action, func(t *testing.T) {
			command := Mutation{Action: action, ID: "22222222-2222-4222-8222-222222222222", ExpectedRevision: 1}
			if action == "approve" {
				command.Note = "已确认合作"
				command.CooperationConfirmed = true
			}
			if err := validateMutation(command, true); !errors.Is(err, ErrInvalid) {
				t.Fatalf("missing market category: got %v, want ErrInvalid", err)
			}
		})
	}
}
