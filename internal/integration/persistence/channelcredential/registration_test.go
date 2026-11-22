package channelcredentialstore

import (
	"context"
	"errors"
	"fmt"
	"testing"

	credential "github.com/henryxu1008/shuomi-ai-commerce/internal/aicapability/channelcredential"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestRegistrationCommitAndOriginalReceipt(t *testing.T) {
	db, store := channelFixture(t)
	ctx := context.Background()
	require.NoError(t, InstallRegistrations(ctx, db))
	input := credential.RegistrationInput{Template: credential.OfficialMiniTemplate, Name: "模型渠道", APIKey: "synthetic-registration-key"}
	command, err := credential.NewRegistrationCommand("org-a", "actor-a", "original", input)
	require.NoError(t, err)

	commit := store.commitTransaction
	store.commitTransaction = func(tx *gorm.DB) error {
		require.NoError(t, tx.Commit().Error)
		return errors.New("synthetic commit reply loss")
	}
	_, err = store.CreateRegistration(ctx, command)
	require.ErrorIs(t, err, credential.ErrOutcomeUnknown)
	store.commitTransaction = commit

	receipt, err := store.RegistrationReceipt(ctx, command.OrganizationID, command.ActorID, command.Key)
	require.NoError(t, err)
	require.Equal(t, command.Registration, receipt.Registration)
	require.Equal(t, command.ConnectionID, receipt.Connection.ConnectionID)
	require.Equal(t, int64(1), receipt.Connection.Revision)
	replayed, err := store.CreateRegistration(ctx, command)
	require.NoError(t, err)
	require.Equal(t, receipt, replayed)

	changed := command
	changed.Registration.Name = "其他名称"
	_, err = store.CreateRegistration(ctx, changed)
	require.ErrorIs(t, err, credential.ErrConflict)
	_, err = store.Receipt(ctx, command.OrganizationID, command.ActorID, command.Key)
	require.ErrorIs(t, err, credential.ErrNotFound)
	_, err = store.RegistrationReceipt(ctx, "org-b", command.ActorID, command.Key)
	require.ErrorIs(t, err, credential.ErrNotFound)
	var count int64
	require.NoError(t, db.Model(&ConnectionRow{}).Count(&count).Error)
	require.Equal(t, int64(1), count)

	renamed, err := store.RenameRegistration(ctx, credential.RenameRegistrationCommand{
		OrganizationID: command.OrganizationID, ActorID: command.ActorID, RequestKey: "rename", ConnectionID: command.ConnectionID,
		Input: credential.RenameRegistrationInput{Name: "新名称", ExpectedRevision: 1},
	})
	require.NoError(t, err)
	require.Equal(t, int64(2), renamed.Revision)
	metadata, err := store.Metadata(ctx, command.OrganizationID, command.ConnectionID)
	require.NoError(t, err)
	require.Equal(t, int64(1), metadata.Revision)

	create := func(key string) error {
		c, err := credential.NewRegistrationCommand(command.OrganizationID, command.ActorID, key, input)
		if err != nil {
			return err
		}
		_, err = store.CreateRegistration(ctx, c)
		return err
	}
	for i := 1; i < 127; i++ {
		require.NoError(t, create(fmt.Sprintf("fixture-%d", i)))
	}
	outcomes := make(chan error, 2)
	go func() { outcomes <- create("capacity-a") }()
	go func() { outcomes <- create("capacity-b") }()
	e1, e2 := <-outcomes, <-outcomes
	if e1 != nil {
		e1, e2 = e2, e1
	}
	require.NoError(t, e1)
	require.ErrorIs(t, e2, credential.ErrRegistrationLimit)
	registrations, err := store.Registrations(ctx, command.OrganizationID)
	require.NoError(t, err)
	require.Len(t, registrations, 128)
	replayed, err = store.CreateRegistration(ctx, command)
	require.NoError(t, err)
	require.Equal(t, receipt, replayed, "capacity does not block the original receipt")
}
