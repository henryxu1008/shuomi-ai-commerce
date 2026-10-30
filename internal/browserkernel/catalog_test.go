package browserkernel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
	"time"
)

func sample() Publish {
	return Publish{Version: "132.0.6834.83", OS: "windows", Arch: "amd64", MinClientVersion: "1.0.0", ArchiveFormat: "zip", Size: 123, SHA256: strings.Repeat("a", 64), DownloadURL: "https://assets.example.test/kernels/132.zip"}
}
func rid(n int) string { return fmt.Sprintf("00000000-0000-4000-8000-%012d", n) }

type testRepository struct {
	rows  []Release
	err   error
	calls int
}

func (r *testRepository) Candidates(_ context.Context, os, arch, after string, limit int) ([]Release, error) {
	r.calls++
	if r.err != nil {
		return nil, r.err
	}
	out := []Release{}
	for _, v := range r.rows {
		if v.ReleaseID > after && v.OS == os && v.Arch == arch && v.WithdrawnAt == nil {
			out = append(out, v)
			if len(out) == limit {
				break
			}
		}
	}
	return out, nil
}
func (r *testRepository) Find(_ context.Context, id string) (Release, error) {
	for _, v := range r.rows {
		if v.ReleaseID == id {
			return v, r.err
		}
	}
	return Release{}, ErrNotFound
}
func (r *testRepository) Publish(context.Context, string, Publish, string) (Release, error) {
	r.calls++
	return Release{}, r.err
}
func (r *testRepository) Withdraw(context.Context, string, string) (Release, error) {
	r.calls++
	return Release{}, r.err
}
func (r *testRepository) Recommend(context.Context, string, bool, string) (Release, error) {
	r.calls++
	return Release{}, r.err
}

func TestCatalogPagingDoesNotConsumeLookahead(t *testing.T) {
	r := &testRepository{}
	for i := 1; i <= 5; i++ {
		r.rows = append(r.rows, Release{ReleaseID: rid(i), Publish: sample()})
	}
	s, e := New(r, []string{"https://assets.example.test"})
	require.NoError(t, e)
	q := Query{OS: "windows", Arch: "amd64", ClientVersion: "1.0.0", Limit: 2}
	var got []string
	for {
		p, e := s.List(context.Background(), q)
		require.NoError(t, e)
		for _, v := range p.Releases {
			got = append(got, v.ReleaseID)
		}
		if p.NextCursor == "" {
			break
		}
		q.Cursor = p.NextCursor
	}
	require.Equal(t, []string{rid(1), rid(2), rid(3), rid(4), rid(5)}, got)
	r.rows[0].MinClientVersion = "2.0.0"
	r.rows[1].MinClientVersion = "2.0.0"
	q.Cursor = ""
	p, e := s.List(context.Background(), q)
	require.NoError(t, e)
	require.Empty(t, p.Releases)
	require.Equal(t, rid(2), p.NextCursor)
	q.Cursor = p.NextCursor
	p, e = s.List(context.Background(), q)
	require.NoError(t, e)
	require.Equal(t, rid(3), p.Releases[0].ReleaseID)
	raw, _ := json.Marshal(p)
	require.NotContains(t, string(raw), "download")
	require.NotContains(t, string(raw), "sha256")
}
func TestCatalogCompatibilityAndTrustedArtifact(t *testing.T) {
	p := sample()
	p.MaxClientVersionExclusive = "2.0.0"
	r := &testRepository{rows: []Release{{ReleaseID: rid(1), Publish: p}}}
	s, e := New(r, []string{"https://assets.example.test", "https://cdn.example.test"})
	require.NoError(t, e)
	q := Query{OS: "windows", Arch: "amd64", ClientVersion: "1.0.0"}
	a, e := s.Artifact(context.Background(), rid(1), q)
	require.NoError(t, e)
	require.Equal(t, p.DownloadURL, a.DownloadURL)
	require.Equal(t, []string{"https://assets.example.test", "https://cdn.example.test"}, a.DownloadOrigins)
	a.DownloadOrigins[0] = "https://evil.test"
	a, e = s.Artifact(context.Background(), rid(1), q)
	require.NoError(t, e)
	require.Equal(t, "https://assets.example.test", a.DownloadOrigins[0])
	for _, v := range []string{"0.9.9", "1.0.0-rc.1", "2.0.0"} {
		q.ClientVersion = v
		_, e = s.Artifact(context.Background(), rid(1), q)
		require.ErrorIs(t, e, ErrNotFound)
	}
	q.ClientVersion = "1.0.0+build.3"
	_, e = s.Artifact(context.Background(), rid(1), q)
	require.NoError(t, e)
	q.OS = "linux"
	_, e = s.Artifact(context.Background(), rid(1), q)
	require.ErrorIs(t, e, ErrNotFound)
	q.OS = "windows"
	now := time.Now()
	r.rows[0].WithdrawnAt = &now
	_, e = s.Artifact(context.Background(), rid(1), q)
	require.ErrorIs(t, e, ErrNotFound)
	r.rows[0].WithdrawnAt = nil
	r.rows[0].DownloadURL = "https://evil.test/x.zip"
	_, e = s.Artifact(context.Background(), rid(1), q)
	require.ErrorIs(t, e, ErrUnavailable)
}
func TestCatalogRejectsInvalidVersionsAndMetadataBeforeRepository(t *testing.T) {
	r := &testRepository{}
	s, e := New(r, []string{"https://assets.example.test"})
	require.NoError(t, e)
	for _, v := range []string{"1", "1.2", "v1.2.3", "01.2.3", "1.2.3.4", "1.2.3-01", "1.2.3+", " 1.2.3"} {
		p := sample()
		p.MinClientVersion = v
		_, e = s.Publish(context.Background(), rid(1), p, "actor")
		require.ErrorIs(t, e, ErrInvalid, v)
		_, e = s.List(context.Background(), Query{OS: "windows", Arch: "amd64", ClientVersion: v, Limit: 1})
		require.ErrorIs(t, e, ErrInvalid, v)
	}
	for _, mutate := range []func(*Publish){func(p *Publish) { p.MaxClientVersionExclusive = "1.0.0" }, func(p *Publish) { p.MaxClientVersionExclusive = "0.9.0" }, func(p *Publish) { p.DownloadURL += "?token=secret" }, func(p *Publish) { p.DownloadURL = "http://assets.example.test/x.zip" }, func(p *Publish) { p.DownloadURL = "https://user@assets.example.test/x.zip" }, func(p *Publish) { p.DownloadURL = "https://assets.example.test.evil/x.zip" }, func(p *Publish) { p.Size = 2147483649 }, func(p *Publish) { p.Size = 0 }, func(p *Publish) { p.SHA256 = strings.Repeat("A", 64) }, func(p *Publish) { p.ArchiveFormat = "exe" }, func(p *Publish) { p.Version = " bad " }} {
		p := sample()
		mutate(&p)
		_, e = s.Publish(context.Background(), rid(1), p, "actor")
		require.ErrorIs(t, e, ErrInvalid)
	}
	_, e = s.Publish(context.Background(), strings.ToUpper("aaaa0000-0000-4000-8000-000000000001"), sample(), "actor")
	require.ErrorIs(t, e, ErrInvalid)
	require.Zero(t, r.calls)
	for _, origins := range [][]string{nil, {"https://assets.example.test/"}, {"https://ASSETS.example.test"}, {"https://assets.example.test:443"}, {"https://:444"}, {"https://assets.example.test:"}, {"https://assets.example.test:0"}, {"https://assets.example.test:65536"}, {"https://assets.example.test:0444"}, {"https://*.example.test"}, {"https://assets.example.test?x=y"}, {"https://assets.example.test", "https://assets.example.test"}} {
		_, e = New(r, origins)
		require.ErrorIs(t, e, ErrInvalid)
	}
	r.err = errors.New("password=secret")
	_, e = s.List(context.Background(), Query{OS: "windows", Arch: "amd64", ClientVersion: "1.0.0", Limit: 1})
	require.ErrorIs(t, e, ErrUnavailable)
	require.NotContains(t, e.Error(), "secret")
}
