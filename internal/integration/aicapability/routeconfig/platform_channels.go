package routeconfiguration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"slices"
	"sort"

	credential "task-processor/internal/aicapability/channelcredential"
	d "task-processor/internal/aicapability/routeconfig"
)

type PlatformChannels struct {
	MaintainerOrganization, ProjectID string
	Reader                            d.CurrentReader
	Writer                            d.Repository
	Catalog                           *SharedCatalog
	Management                        *credential.Management
}

type PlatformChannelView struct {
	Connection  credential.ManagedConnection `json:"connection"`
	Publication *d.PlatformChannel           `json:"publication,omitempty"`
	Revision    string                       `json:"revision"`
}

type PlatformPublishInput struct {
	ExpectedRevision string            `json:"expectedRevision"`
	Channel          d.PlatformChannel `json:"channel"`
}

func (s *PlatformChannels) read(ctx context.Context, connection string) (d.PlatformChannel, string, error) {
	scope := d.PlatformChannelScope(connection)
	r, err := s.Reader.Current(ctx, scope)
	if err != nil {
		return d.PlatformChannel{}, "", err
	}
	p := r.Payload.PlatformChannel
	if r.Scope != scope || r.SchemaVersion != 1 || p == nil || p.Validate(connection) != nil {
		return d.PlatformChannel{}, "", d.ErrUnavailable
	}
	return *p, r.ID, nil
}

func (s *PlatformChannels) CatalogOrganizations() []string {
	orgs := []string{}
	seen := map[string]bool{}
	for scope := range s.Catalog.Catalog.definitions {
		org := scope.OrganizationID
		if !seen[org] && s.Catalog.Managed.Supports(org, credential.OfficialMiniTemplate) {
			seen[org] = true
			orgs = append(orgs, org)
		}
	}
	sort.Strings(orgs)
	return orgs
}

func (s *PlatformChannels) List(ctx context.Context) ([]PlatformChannelView, error) {
	rows, err := s.Management.RegisteredConnections(ctx, credential.PlatformOwner)
	if err != nil {
		return nil, err
	}
	out := make([]PlatformChannelView, 0, len(rows))
	for _, row := range rows {
		v := PlatformChannelView{Connection: row, Revision: d.NoRevision}
		publication, rev, err := s.read(ctx, row.ConnectionID)
		if err == nil {
			v.Publication, v.Revision = &publication, rev
		} else if !errors.Is(err, d.ErrNotFound) {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

func (s *PlatformChannels) Publish(ctx context.Context, actor, key string, input PlatformPublishInput) (d.Revision, error) {
	release := input.Channel
	if release.Validate(release.ConnectionID) != nil || !d.ValidFact(actor, 128) || !d.ValidFact(key, 128) || !d.ValidFact(input.ExpectedRevision, 128) {
		return d.Revision{}, d.ErrInvalid
	}
	ident, err := json.Marshal(struct {
		Actor string
		Input PlatformPublishInput
	}{actor, input})
	if err != nil {
		return d.Revision{}, d.ErrInvalid
	}
	digest := sha256.Sum256(ident)
	scope := d.PlatformChannelScope(release.ConnectionID)
	return s.Writer.Publish(ctx, d.Command{Scope: scope, ActorID: actor, Key: key, Fingerprint: hex.EncodeToString(digest[:]), Input: d.Input{ExpectedRevision: input.ExpectedRevision}},
		func(tx d.CredentialReader, _ string) (d.Payload, error) {
			locker, ok := tx.(interface {
				LockPlatformSelection(context.Context, string) error
			})
			if !ok {
				return d.Payload{}, d.ErrUnavailable
			}
			if err := locker.LockPlatformSelection(ctx, release.ConnectionID); err != nil {
				return d.Payload{}, err
			}
			if _, err := s.Catalog.Managed.Registrations.Registration(ctx, credential.PlatformOwner, release.ConnectionID); err != nil {
				return d.Payload{}, d.ErrNotFound
			}
			for _, org := range release.Organizations {
				resolved, err := s.resolve(ctx, org, release, "")
				if err != nil {
					return d.Payload{}, err
				}
				q, err := s.Catalog.qualifyDefinitions(ctx, org, d.ModelCategoryText, release.Reference(), resolved)
				if err != nil {
					return d.Payload{}, err
				}
				if !q.Ready {
					return d.Payload{}, d.ErrNeedsConfiguration
				}
			}
			return d.Payload{PlatformChannel: &release}, nil
		})
}

func (s *PlatformChannels) Receipt(ctx context.Context, connection, actor, key string) (d.Revision, error) {
	if !credential.IsRegisteredConnection(connection) || !d.ValidFact(key, 128) {
		return d.Revision{}, d.ErrInvalid
	}
	r, err := s.Writer.Receipt(ctx, d.PlatformChannelScope(connection), key)
	if err == nil && (r.ActorID != actor || r.Payload.PlatformChannel == nil || r.Payload.PlatformChannel.Validate(connection) != nil) {
		return d.Revision{}, d.ErrNotFound
	}
	return r, err
}

func (s *PlatformChannels) references(ctx context.Context, org string) ([]d.ModelBindingReference, error) {
	rows, err := s.Catalog.Managed.Registrations.Registrations(ctx, credential.PlatformOwner)
	if err != nil {
		return nil, d.ErrUnavailable
	}
	refs := []d.ModelBindingReference{}
	for _, row := range rows {
		p, _, err := s.read(ctx, row.ConnectionID)
		if errors.Is(err, d.ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if p.Allows(org, p.Reference()) {
			refs = append(refs, p.Reference())
		}
	}
	return refs, nil
}

func (s *PlatformChannels) admitted(ctx context.Context, org string, ref d.ModelBindingReference) (modelDefinitions, error) {
	hidden := modelDefinitions{owner: credential.PlatformOwner, reason: "PLATFORM_ACCESS_WITHDRAWN"}
	p, rev, err := s.read(ctx, d.PlatformConnection(ref.ID))
	if errors.Is(err, d.ErrNotFound) {
		return hidden, nil
	}
	if err != nil {
		return modelDefinitions{}, err
	}
	if !slices.Contains(p.Organizations, org) {
		return hidden, nil
	}
	if ref != p.Reference() {
		hidden.reason = "PLATFORM_BINDING_REPLACED"
		return hidden, nil
	}
	return s.resolve(ctx, org, p, rev)
}

func (s *PlatformChannels) resolve(ctx context.Context, org string, p d.PlatformChannel, version string) (modelDefinitions, error) {
	managed := s.Catalog.Managed
	reg, err := managed.Registrations.Registration(ctx, credential.PlatformOwner, p.ConnectionID)
	if err != nil || reg.OrganizationID != credential.PlatformOwner || reg.ConnectionID != p.ConnectionID {
		return modelDefinitions{}, d.ErrUnavailable
	}
	m, err := managed.Metadata.Metadata(ctx, credential.PlatformOwner, p.ConnectionID)
	if err != nil || m.OrganizationID != credential.PlatformOwner || m.ConnectionID != p.ConnectionID {
		return modelDefinitions{}, d.ErrUnavailable
	}
	result := modelDefinitions{owner: credential.PlatformOwner, label: reg.Name, publication: version}
	if !m.Enabled || !m.KeyPresent {
		result.reason = "CHANNEL_DISABLED"
		return result, nil
	}
	result.values, err = managed.templates(org, reg.Template, m.BaseURL, p.ConnectionID, p.Reference().ID)
	if errors.Is(err, d.ErrNeedsConfiguration) {
		result.reason = "OPEN_CONSUMER_MISSING"
		return result, nil
	}
	if err != nil {
		return modelDefinitions{}, err
	}
	for i := range result.values {
		result.values[i].ReferenceVersion = p.Reference().Version
		result.values[i].PriceID = p.PriceID
		result.values[i].CredentialOwner = credential.PlatformOwner
	}
	return result, nil
}
