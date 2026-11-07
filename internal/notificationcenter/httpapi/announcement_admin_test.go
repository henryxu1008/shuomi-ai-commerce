package notificationhttp

import (
	"testing"

	"task-processor/internal/authz"
	"task-processor/internal/httproute"
)

func TestAnnouncementAdminReadsRequireVerifiedPlatformAuthority(t *testing.T) {
	for _, path := range []string{"/api/v1/platform/notifications/official", "/api/v1/platform/notifications/official/:id"} {
		t.Run(path, func(t *testing.T) {
			for _, route := range Routes(&Handler{}) {
				if route.Path != path || route.Method != "GET" {
					continue
				}
				if route.Permission != authz.PermissionListingKitPlatformAdm || route.AuthPolicy != httproute.AuthPolicyCurrentIdentityWithVerifiedRoles || route.OrganizationAccessPolicy != httproute.OrganizationAccessPolicyNone || !route.RejectUnreadRequestBody {
					t.Fatalf("wrong policy: %+v", route)
				}
				if err := ValidateDescriptor(route); err != nil {
					t.Fatal(err)
				}
				route.Permission = ""
				if err := ValidateDescriptor(route); err == nil {
					t.Fatal("accepted an unguarded announcement history route")
				}
				return
			}
			t.Fatal("announcement management read route missing")
		})
	}
}
