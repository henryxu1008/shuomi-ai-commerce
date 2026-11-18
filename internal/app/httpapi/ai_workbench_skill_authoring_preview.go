package httpapi

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"task-processor/internal/aicapability"
	"task-processor/internal/aiworkbench"
	wbstore "task-processor/internal/integration/persistence/aiworkbench"
	"task-processor/internal/skill"
)

func authoringProfileFormat(profile aicapability.ModelProfile) int {
	switch {
	case profile.PromptVersion == "ai-workbench-skill-authoring-v1" && profile.OutputSchemaVersion == "personal-title-skill-draft-v1":
		return 1
	case profile.PromptVersion == "ai-workbench-skill-authoring-v2" && profile.OutputSchemaVersion == "personal-title-skill-draft-v2":
		return 2
	default:
		return 0
	}
}

func authoringReleaseFormat(version uint64) int {
	switch version {
	case 1:
		return 1
	case 2, skill.GeneralAuthoringVersion:
		return 2
	default:
		return 0
	}
}

type authoringPreviewInput struct {
	Requirement  string           `json:"requirement"`
	SkillVersion uint64           `json:"skillVersion"`
	SkillDigest  string           `json:"skillDigest"`
	Root         string           `json:"root,omitempty"`
	BaseRevision string           `json:"baseRevision,omitempty"`
	Snapshot     *titleSkillDraft `json:"snapshot,omitempty"`
}

type authoringConfirmInput struct {
	authoringPreviewInput
	Quote string `json:"quote"`
}

const authoringQuoteLifetime = 5 * time.Minute

// The server issues each preview its own UUIDv7. Its immutable timestamp
// determines expiry even when the quote signing key changes or a caller lies.
func authoringQuoteDeadline(key string) (time.Time, error) {
	id, err := uuid.Parse(key)
	if err != nil || id.Version() != 7 || id.Variant() != uuid.RFC4122 || id.String() != key {
		return time.Time{}, aiworkbench.ErrInvalid
	}
	seconds, nanos := id.Time().UnixTime()
	return time.Unix(seconds, nanos).UTC().Add(authoringQuoteLifetime), nil
}

type authoringSaveInput struct {
	Revision string          `json:"revision"`
	Draft    titleSkillDraft `json:"draft"`
}

func newAuthoringRequest(input authoringPreviewInput, conversation, member, key string, scope aiworkbench.Scope) (wbstore.AuthoringDraft, error) {
	if !acquisitionHTTPUUID(conversation) || !acquisitionHTTPUUID(key) || strings.TrimSpace(input.Requirement) == "" ||
		len(input.Requirement) > 8192 || authoringReleaseFormat(input.SkillVersion) == 0 || len(input.SkillDigest) != 64 {
		return wbstore.AuthoringDraft{}, aiworkbench.ErrInvalid
	}
	row := wbstore.AuthoringDraft{
		OrganizationID: scope.OrganizationID, ActorID: scope.ActorID, ConversationID: conversation, Key: key, MemberID: member,
		Requirement: input.Requirement, SkillVersion: input.SkillVersion, SkillDigest: input.SkillDigest,
		InvocationID: authoringInvocation(scope, conversation, key), FormatVersion: authoringReleaseFormat(input.SkillVersion),
	}
	if input.Root == "" && input.BaseRevision == "" && input.Snapshot == nil {
		return row, nil
	}
	if !acquisitionHTTPUUID(input.Root) || !acquisitionHTTPUUID(input.BaseRevision) || input.Root == key ||
		input.Snapshot == nil || row.FormatVersion != 2 || (input.SkillVersion == skill.GeneralAuthoringVersion && input.Snapshot.Script != nil) {
		return row, aiworkbench.ErrInvalid
	}
	// Revision prompts may include an unfinished manually edited snapshot.
	// The confirmed output and save still require a valid complete package.
	if s := input.Snapshot.Script; s != nil && (len(s.Source) > skill.MaxFileBytes || len(s.Description) > 1536) {
		return row, aiworkbench.ErrInvalid
	}
	raw, err := json.Marshal(input.Snapshot)
	if err != nil || len(raw) > 65536 {
		return row, aiworkbench.ErrInvalid
	}
	row.RootKey, row.BaseRevision, row.InputSnapshot = &input.Root, &input.BaseRevision, string(raw)
	return row, nil
}

func (p *workbenchSkillAuthoring) signAuthoringQuote(row wbstore.AuthoringDraft) string {
	// These claims bind the public quote to one scope, payload, and route.
	// Length prefixes from JSON prevent concatenation ambiguity.
	canonical, _ := json.Marshal([]any{row.OrganizationID, row.ActorID, row.MemberID, row.ConversationID, row.Key,
		row.ExpiresAt.UnixMilli(), row.SkillVersion, row.SkillDigest, row.InputHash, row.ProfileDigest, row.MaximumPoints,
		row.Requirement, row.InputSnapshot, row.RootKey, row.BaseRevision})
	mac := hmac.New(sha256.New, p.quoteKey[:])
	_, _ = mac.Write(canonical)
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (a *aiWorkbenchApplication) previewSkillAuthoring(c *gin.Context, ctx context.Context, scope aiworkbench.Scope, conversation, member string) error {
	var input authoringPreviewInput
	if workbenchJSONLimit(c, &input, skill.MaxRequestBytes) != nil {
		return aiworkbench.ErrInvalid
	}
	previewID, err := uuid.NewV7()
	if err != nil {
		return aiworkbench.ErrUnavailable
	}
	key := previewID.String()
	row, err := newAuthoringRequest(input, conversation, member, key, scope)
	if err != nil {
		return err
	}
	if err := a.store.CheckAuthoringBase(ctx, scope, row); err != nil {
		return err
	}
	_, estimate, points, _, err := a.authoring.prepare(ctx, scope, row)
	if err != nil {
		return err
	}
	row.InputHash, row.ProfileDigest, row.MaximumPoints = estimate.InputHash, estimate.ProfileDigest, points
	row.ExpiresAt, err = authoringQuoteDeadline(key)
	if err != nil {
		return err
	}
	workbenchReply(c, 200, gin.H{"key": key, "expiresAt": row.ExpiresAt, "quote": a.authoring.signAuthoringQuote(row)})
	return nil
}

func (a *aiWorkbenchApplication) confirmSkillAuthoring(c *gin.Context, ctx context.Context, scope aiworkbench.Scope, conversation, member string) (wbstore.AuthoringDraft, error) {
	var input authoringConfirmInput
	if workbenchJSONLimit(c, &input, skill.MaxRequestBytes) != nil {
		return wbstore.AuthoringDraft{}, aiworkbench.ErrInvalid
	}
	row, err := newAuthoringRequest(input.authoringPreviewInput, conversation, member, c.Param("authoring_key"), scope)
	if err != nil {
		return row, err
	}
	original, err := a.store.AuthoringDraft(ctx, scope, conversation, row.Key)
	if err == nil {
		if !original.MatchesAuthoringInput(row) {
			return row, aiworkbench.ErrIdempotencyConflict
		}
		return a.generateSkillAuthoring(ctx, scope, original)
	}
	if !errors.Is(err, aiworkbench.ErrNotFound) {
		return row, err
	}
	now := time.Now().UTC()
	row.ExpiresAt, err = authoringQuoteDeadline(row.Key)
	if err != nil || len(input.Quote) != 43 || row.ExpiresAt.After(now.Add(authoringQuoteLifetime+time.Second)) {
		return row, aiworkbench.ErrInvalid
	}
	model, estimate, points, _, err := a.authoring.prepare(ctx, scope, row)
	if err != nil {
		return row, err
	}
	row.InputHash, row.ProfileDigest, row.MaximumPoints = estimate.InputHash, estimate.ProfileDigest, points
	if !hmac.Equal([]byte(input.Quote), []byte(a.authoring.signAuthoringQuote(row))) {
		return row, aiworkbench.ErrProposalStale
	}
	deadline := now.Add(model.Profile.DeadlineBound)
	row.Deadline = &deadline
	row, err = a.store.StartAuthoring(ctx, scope, row)
	if err != nil {
		return row, err
	}
	return a.generateSkillAuthoring(ctx, scope, row)
}

func (a *aiWorkbenchApplication) recoverSkillGeneration(c *gin.Context, ctx context.Context, scope aiworkbench.Scope, conversation, member string) (wbstore.AuthoringDraft, error) {
	key := c.Param("authoring_key")
	expires, err := authoringQuoteDeadline(key)
	if workbenchEmptyBody(c) != nil || err != nil {
		return wbstore.AuthoringDraft{}, aiworkbench.ErrInvalid
	}
	row, err := a.store.AuthoringConfirmation(ctx, scope, conversation, key, expires)
	if err != nil {
		return row, err
	}
	if row.MemberID != member {
		return row, aiworkbench.ErrNotFound
	}
	return a.generateSkillAuthoring(ctx, scope, row)
}

func (a *aiWorkbenchApplication) freezeSkillPackage(c *gin.Context, ctx context.Context, scope aiworkbench.Scope, row wbstore.AuthoringDraft) (wbstore.AuthoringDraft, error) {
	var input authoringSaveInput
	key, err := workbenchKey(c)
	if err != nil || workbenchJSONLimit(c, &input, skill.MaxRequestBytes) != nil || !acquisitionHTTPUUID(input.Revision) ||
		row.RootKey != nil || row.State != "COMPLETE" || (row.FormatVersion != 2 && input.Draft.Script != nil && row.CurrentRevision == nil) {
		return row, aiworkbench.ErrInvalid
	}
	pkg, err := input.Draft.skillPackageFor(authoredTarget(row.SkillVersion))
	if err != nil {
		return row, err
	}
	files, err := json.Marshal(pkg.Files)
	if err != nil {
		return row, aiworkbench.ErrInvalid
	}
	return a.store.FreezeAuthoringPackage(ctx, scope, row, input.Revision, key, pkg.Digest, string(files), 2)
}

func authoringRootKey(row wbstore.AuthoringDraft) string {
	if row.RootKey != nil {
		return *row.RootKey
	}
	return row.Key
}
