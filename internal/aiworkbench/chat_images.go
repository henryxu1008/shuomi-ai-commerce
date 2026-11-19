package aiworkbench

import (
	"encoding/json"
	"strconv"

	"task-processor/internal/aicapability"
	"task-processor/internal/ledger/storagecapacity"
)

// ChatImage is the frozen, bounded content identity in a Workbench command.
// Byte content and object names are never accepted as WorkScope fields.
type ChatImage struct {
	SHA256 string `json:"sha256"`
	MIME   string `json:"mime"`
	Bytes  int    `json:"bytes"`
	Width  int    `json:"w"`
	Height int    `json:"h"`
}

func (i ChatImage) ModelInput(selection string) aicapability.SingleImageInput {
	return aicapability.SingleImageInput{SelectionDigest: selection, ContentSHA256: i.SHA256,
		MediaType: i.MIME, ByteLength: i.Bytes, Width: i.Width, Height: i.Height}
}

func ValidChatImageSelection(images []ChatImage) bool {
	if len(images) == 0 || len(images) > aicapability.MaximumChatImages {
		return false
	}
	identities := make([]aicapability.SingleImageInput, len(images))
	for n, image := range images {
		identities[n] = image.ModelInput(image.SHA256)
	}
	return aicapability.ValidChatImages(identities)
}

// ChatImageOperation identifies one batch at both the storage and message
// boundaries. Members and conversations cannot share an object namespace.
func ChatImageOperation(scope Scope, member, conversation, request string) string {
	identity, _ := json.Marshal([]string{scope.OrganizationID, scope.ActorID, member, conversation, request})
	return storagecapacity.Digest(string(identity))
}

func ChatImageObject(operation string, index int) string {
	return "chat-image/" + operation + "/" + strconv.Itoa(index)
}
