package httpapi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"image"
	"time"

	"task-processor/internal/aicapability"
	"task-processor/internal/aiworkbench"
	governed "task-processor/internal/integration/aicapability/einomodel"
	"task-processor/internal/integration/s3"
	"task-processor/internal/ledger/storagecapacity"
)

const chatImageCapacityOwner = "workbench_chat_image"

var errChatImageStorageUnknown = errors.New("chat image storage outcome unknown")

// The runtime supplies a private uploader and its verified capacity namespace.
// No caller can choose the namespace, key, or capacity classification.
type ChatImageStorage struct {
	Objects interface {
		PutImmutable(context.Context, s3.ImmutableObjectPut) error
		ReadObject(context.Context, string, int64) ([]byte, s3.ObjectInspection, error)
	}
	Capacity  storagecapacity.Writer
	Namespace string
}

func inspectChatImages(group [][]byte) ([]aiworkbench.ChatImage, error) {
	if len(group) < 1 || len(group) > aicapability.MaximumChatImages {
		return nil, aiworkbench.ErrInvalid
	}
	total := 0
	for _, data := range group {
		total += len(data)
		if len(data) == 0 || len(data) > aicapability.MaximumSingleImageBytes || total > aicapability.MaximumChatImageBytes {
			return nil, aiworkbench.ErrInvalid
		}
	}
	result := make([]aiworkbench.ChatImage, len(group))
	for n, data := range group {
		cfg, kind, err := image.DecodeConfig(bytes.NewReader(data))
		if err != nil || kind != "jpeg" && kind != "png" {
			return nil, aiworkbench.ErrInvalid
		}
		hash := sha256.Sum256(data)
		picture := aiworkbench.ChatImage{SHA256: hex.EncodeToString(hash[:]), MIME: "image/" + kind, Bytes: len(data), Width: cfg.Width, Height: cfg.Height}
		if governed.ValidateSingleImageContent(picture.ModelInput(picture.SHA256), data) != nil {
			return nil, aiworkbench.ErrInvalid
		}
		result[n] = picture
	}
	return result, nil
}

func (s *ChatImageStorage) available() bool {
	return s != nil && s.Objects != nil && s.Capacity != nil && storagecapacity.ValidID(s.Namespace, 256)
}

// Only an authenticated consumer supplies operation. A conditional PUT can
// race another recovery, so only a verified original body can finish a slot.
func (s *ChatImageStorage) read(ctx context.Context, operation string, index int, image aiworkbench.ChatImage) ([]byte, bool, error) {
	if !s.available() {
		return nil, false, aiworkbench.ErrUnavailable
	}
	data, info, err := s.Objects.ReadObject(ctx, aiworkbench.ChatImageObject(operation, index), int64(image.Bytes))
	if err != nil {
		return nil, false, aiworkbench.ErrUnavailable
	}
	if !info.Exists {
		return nil, false, nil
	}
	if info.ContentType != image.MIME || info.ContentLength != int64(image.Bytes) ||
		governed.ValidateSingleImageContent(image.ModelInput(image.SHA256), data) != nil {
		return nil, false, aiworkbench.ErrUnavailable
	}
	return data, true, nil
}

func (s *ChatImageStorage) store(ctx context.Context, org, operation, question string, pictures []aiworkbench.ChatImage, data [][]byte) error {
	if !s.available() || !aiworkbench.ValidChatImageSelection(pictures) || len(data) != len(pictures) {
		return aiworkbench.ErrUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	wire, _ := json.Marshal(struct {
		Question string
		Pictures []aiworkbench.ChatImage
	}{question, pictures})
	fingerprint := storagecapacity.Digest(string(wire))
	objects := make([]storagecapacity.Object, len(pictures))
	var total int64
	for n, picture := range pictures {
		if governed.ValidateSingleImageContent(picture.ModelInput(picture.SHA256), data[n]) != nil {
			return aiworkbench.ErrInvalid
		}
		objects[n] = storagecapacity.Object{Namespace: s.Namespace, Key: aiworkbench.ChatImageObject(operation, n),
			SHA256: picture.SHA256, SizeBytes: int64(picture.Bytes), Category: "business_attachment"}
		total += objects[n].SizeBytes
	}
	reservation, err := s.Capacity.ReserveWrite(ctx, storagecapacity.WriteIntent{OrganizationID: org, Owner: chatImageCapacityOwner,
		OperationID: operation, Fingerprint: fingerprint, MaxBytes: total, MaxObjects: len(objects)})
	if err != nil {
		return err
	}
	if s.Capacity.BindObjects(ctx, org, reservation.ID, objects) != nil || s.Capacity.ClaimDispatch(ctx, org, reservation.ID) != nil {
		return errChatImageStorageUnknown
	}
	for n, object := range objects {
		_, found, err := s.read(ctx, operation, n, pictures[n])
		if err != nil {
			return errChatImageStorageUnknown
		}
		if !found {
			_ = s.Objects.PutImmutable(ctx, s3.ImmutableObjectPut{Key: object.Key, Data: data[n], ContentType: pictures[n].MIME,
				SHA256: object.SHA256, SizeBytes: object.SizeBytes})
			_, found, err = s.read(ctx, operation, n, pictures[n])
		}
		if err != nil || !found ||
			s.Capacity.ConfirmStored(ctx, org, reservation.ID, object.Namespace, object.Key, object.SHA256, object.SizeBytes) != nil {
			return errChatImageStorageUnknown
		}
	}
	proof := storagecapacity.CloseProof{Owner: chatImageCapacityOwner, OperationID: operation,
		Fingerprint: fingerprint, TerminalReceipt: "chat-image:" + operation}
	if s.Capacity.CloseWrite(ctx, org, reservation.ID, proof) != nil {
		return errChatImageStorageUnknown
	}
	return nil
}
