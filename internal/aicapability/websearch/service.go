package websearch

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"time"
	"unicode"

	credential "task-processor/internal/aicapability/channelcredential"
)

const OwnedDailyLimit = 30

func (s *Service) Credential(ctx context.Context, owner string) (credential.Metadata, error) {
	m, err := s.Metadata.Metadata(ctx, owner, Connection)
	if errors.Is(err, credential.ErrNotFound) {
		return credential.Metadata{}, nil
	}
	if err != nil {
		return credential.Metadata{}, err
	}
	if m.OrganizationID != owner || m.ConnectionID != Connection || m.Provider != "brave" || m.Channel != Connection || m.BaseURL != BaseURL {
		return credential.Metadata{}, ErrUnavailable
	}
	return m, nil
}

func (s *Service) SaveCredential(ctx context.Context, owner, actor, requestKey string, revision int64, key string, enabled bool) (credential.Metadata, error) {
	if strings.IndexFunc(key, func(r rune) bool { return r < 33 || r > unicode.MaxASCII }) >= 0 {
		return credential.Metadata{}, ErrInvalid
	}
	return s.Writer.PublishSearchCredential(ctx, credential.Command{OrganizationID: owner, ActorID: actor, Key: requestKey,
		ConnectionID: Connection, ExpectedRevision: revision, Provider: "brave", Channel: Connection,
		Account: "default", Region: "global", BaseURL: BaseURL, Secret: key, Enabled: enabled})
}

func (s *Service) CredentialReceipt(ctx context.Context, owner, actor, requestKey string) (credential.Metadata, error) {
	m, err := s.Writer.Receipt(ctx, owner, actor, requestKey)
	if err == nil && (m.OrganizationID != owner || m.ConnectionID != Connection || m.Provider != "brave") {
		return credential.Metadata{}, ErrNotFound
	}
	return m, err
}

func (s *Service) route(ctx context.Context, org, source string, choiceRevision int64) (Route, string, error) {
	r := Route{Owner: org, Source: source, Country: "US", Language: "en", TenantLimit: OwnedDailyLimit, TotalLimit: OwnedDailyLimit}
	label := "企业自有 Brave"
	r.ChoiceRevision = choiceRevision
	switch source {
	case Platform:
		p, err := s.Repo.Publication(ctx)
		if err != nil {
			return Route{}, "", err
		}
		if !p.Published || !slices.Contains(p.Organizations, org) {
			return Route{}, "", ErrNotFound
		}
		r.Owner, label = credential.PlatformOwner, p.Name
		r.Country, r.Language = p.Country, p.Language
		r.TotalLimit, r.TenantLimit = p.TotalLimit, p.TenantLimit
		r.PublicationRevision, r.MaxUSDMicros = p.Revision, p.MaxUSDMicros
	case Owned:
	default:
		return Route{}, "", ErrNotFound
	}
	m, err := s.Credential(ctx, r.Owner)
	if err != nil {
		return Route{}, "", err
	}
	if !m.Enabled || !m.KeyPresent || m.Reference == "" {
		return Route{}, label, ErrNotFound
	}
	r.Credential = m
	r.Preview = s.digest(org, source, choiceRevision, r.PublicationRevision, m.Reference)
	return r, label, nil
}

func (s *Service) View(ctx context.Context, org string) (View, error) {
	current, err := s.Repo.Selection(ctx, org)
	if err != nil {
		return View{}, err
	}
	view := View{Selection: current, Options: []Option{}}
	for _, source := range []string{Platform, Owned} {
		r, label, err := s.route(ctx, org, source, current.Revision)
		if errors.Is(err, ErrNotFound) {
			continue
		}
		if err != nil {
			return View{}, err
		}
		view.Options = append(view.Options, Option{Source: source, Name: label, Available: true, Limit: r.TenantLimit, Preview: r.Preview})
	}
	return view, nil
}

func (s *Service) Select(ctx context.Context, org, actor, key string, selection Selection) (Selection, error) {
	if selection.Revision < 0 || (selection.Source != Platform && selection.Source != Owned) {
		return Selection{}, ErrInvalid
	}
	prior, err := s.Repo.SelectionReceipt(ctx, org, actor, key)
	if err == nil {
		if prior.Source != selection.Source || prior.Revision != selection.Revision+1 {
			return Selection{}, ErrConflict
		}
		return prior, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return Selection{}, err
	}
	if _, _, err := s.route(ctx, org, selection.Source, selection.Revision); err != nil {
		return Selection{}, err
	}
	return s.Repo.SaveSelection(ctx, org, actor, key, selection)
}

func (s *Service) Current(ctx context.Context, org string) (Route, Option, error) {
	choice, err := s.Repo.Selection(ctx, org)
	if err != nil {
		return Route{}, Option{}, err
	}
	r, label, err := s.route(ctx, org, choice.Source, choice.Revision)
	return r, Option{Source: choice.Source, Name: label, Available: err == nil, Limit: r.TenantLimit, Preview: r.Preview}, err
}

func (s *Service) Submit(ctx context.Context, scope Scope, key, query, preview string, authorize func(context.Context) error) (Response, error) {
	if !ValidQuery(query) || len(s.FingerprintKey) < 32 {
		return Response{}, ErrInvalid
	}
	if err := authorize(ctx); err != nil {
		return Response{}, err
	}
	fingerprint := s.digest(scope, key, query, preview)
	record, err := s.Repo.Find(ctx, scope, key, fingerprint)
	if err == nil {
		return Response{Receipt: record}, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return Response{}, err
	}
	route, _, err := s.Current(ctx, scope.Organization)
	if err != nil {
		return Response{}, err
	}
	if preview == "" || preview != route.Preview {
		return Response{}, ErrConflict
	}
	attempt := Attempt{Scope: scope, Route: route, Key: key, Query: query, Fingerprint: fingerprint}
	beforeHold := time.Now()
	receipt, first, limit, err := s.Repo.Hold(ctx, attempt)
	if err != nil || !first {
		return Response{Receipt: receipt}, err
	}
	attempt.Deadline = limit
	// PostgreSQL picks the window. Translate its remaining duration to a
	// conservative local monotonic deadline to tolerate host clock skew.
	send, cancel := context.WithDeadline(ctx, beforeHold.Add(limit.Sub(receipt.StartedAt)))
	defer cancel()

	finish := func(state string, usage *int, results []Result) (Response, error) {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		receipt, err := s.Repo.Finish(ctx, attempt, state, usage)
		if err != nil {
			return Response{}, err
		}
		if err := authorize(ctx); err != nil {
			return Response{}, ErrOutcomeUnknown
		}
		return Response{Receipt: receipt, Results: results}, nil
	}
	zero, one := 0, 1
	if authorize(send) != nil {
		return finish(NotSent, &zero, nil)
	}
	current, _, err := s.Current(send, scope.Organization)
	if err != nil || current.Preview != preview {
		return finish(NotSent, &zero, nil)
	}
	secret, err := s.Secrets.Resolve(send, route.Owner, Connection, route.Credential.Reference)
	if err != nil || secret.APIKey == "" || secret.Metadata != route.Credential {
		return finish(NotSent, &zero, nil)
	}
	claimed, err := s.Repo.Claim(send, attempt)
	if err != nil {
		return Response{}, err
	}
	if !claimed || send.Err() != nil || authorize(send) != nil {
		return finish(NotSent, &zero, nil)
	}
	latest, _, err := s.Current(send, scope.Organization)
	if err != nil || latest.Preview != preview || send.Err() != nil {
		return finish(NotSent, &zero, nil)
	}
	results, err := s.Provider.Search(send, secret.APIKey, query, route.Country, route.Language)
	if err != nil {
		return finish(Unknown, nil, nil)
	}
	status := Complete
	if len(results) == 0 {
		status = NoResults
	}
	return finish(status, &one, results)
}

func (s *Service) digest(values ...any) string {
	payload, _ := json.Marshal(values)
	mac := hmac.New(sha256.New, s.FingerprintKey)
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}
