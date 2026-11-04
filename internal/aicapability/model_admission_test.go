package aicapability

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEinoCommonImageIdentityDoesNotDependOnMiniProtocol(t *testing.T) {
	i := SingleImageInput{SelectionDigest: strings.Repeat("a", 64), ContentSHA256: strings.Repeat("b", 64),
		MediaType: "image/png", ByteLength: 100, Width: 2, Height: 2}
	require.True(t, i.Valid(), "image identity belongs to the image owner, not a model's detail or metering policy")
	i.Width = MaximumSingleImageDimension + 1
	require.False(t, i.Valid(), "separating model parameters must retain the existing resource limits")
}

func TestEinoExplicitWireModelKeepsManufacturerPriceIdentity(t *testing.T) {
	p := testTextProfile()
	p.ModelID = "vendor/model"
	p.OfficialTariff.Model = "model"
	// Decode the proposed original tariff payload, not an alternative price owner.
	raw, err := json.Marshal(p.OfficialTariff)
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(raw, &payload))
	payload["wireModel"] = p.ModelID
	raw, err = json.Marshal(payload)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, &p.OfficialTariff))
	in := TextInputIdentity{OrganizationID: "org", ActorID: "actor", MemberID: "member", Operation: OperationAIWorkbenchChatPlan,
		System: "system", Prompt: "prompt", Profile: p}
	_, err = QuoteText(in)
	require.NoError(t, err, "an explicit wire binding must preserve, rather than rename, the manufacturer-priced model")
	payload["wireModel"] = "different/model"
	raw, err = json.Marshal(payload)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, &in.Profile.OfficialTariff))
	_, err = QuoteText(in)
	require.Error(t, err, "a tariff must never be borrowed by a different wire model")
}
