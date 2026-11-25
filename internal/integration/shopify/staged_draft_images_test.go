package shopify

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const stagedFixture = `{
 "url":"https://shopify-staged-uploads.storage.googleapis.com/",
 "resourceUrl":"https://shopify-staged-uploads.storage.googleapis.com/tmp/123/products/abc/image-1.png",
 "parameters":[
 {"name":"Content-Type","value":"image/png"},
 {"name":"success_action_status","value":"201"},
 {"name":"acl","value":"private"},
 {"name":"key","value":"tmp/123/products/abc/image-1.png"},
 {"name":"x-goog-date","value":"20261125T000000Z"},
 {"name":"x-goog-credential","value":"fixture"},
 {"name":"x-goog-algorithm","value":"GOOG4-RSA-SHA256"},
 {"name":"x-goog-signature","value":"fixture"},
 {"name":"policy","value":"fixture"}]}`

func TestStagedTargetBindsSignedObjectToApprovedFile(t *testing.T) {
	var target stagedImageTarget
	require.NoError(t, json.Unmarshal([]byte(stagedFixture), &target))
	require.True(t, target.valid("image-1.png"))
	require.False(t, target.valid("other.png"))
	for name, change := range map[string]func(*stagedImageTarget){
		"redirect destination": func(v *stagedImageTarget) { v.URL = "https://elsewhere.example/" },
		"arbitrary storage host": func(v *stagedImageTarget) {
			v.ResourceURL = strings.ReplaceAll(v.ResourceURL, stagedHost, "storage.googleapis.com")
		},
		"signed key mismatch":  func(v *stagedImageTarget) { v.ResourceURL = strings.ReplaceAll(v.ResourceURL, "/abc/", "/def/") },
		"duplicate form field": func(v *stagedImageTarget) { v.Parameters[2] = v.Parameters[3] },
	} {
		t.Run(name, func(t *testing.T) {
			var changed stagedImageTarget
			require.NoError(t, json.Unmarshal([]byte(stagedFixture), &changed))
			change(&changed)
			require.False(t, changed.valid("image-1.png"))
		})
	}
}

func TestStagedUploadSendsOnlySignedFormAndImage(t *testing.T) {
	var target stagedImageTarget
	require.NoError(t, json.Unmarshal([]byte(stagedFixture), &target))
	contents := []byte("\x89PNG\r\n\x1a\nfixture")
	client := &http.Client{Transport: transportFunc(func(req *http.Request) (*http.Response, error) {
		require.Equal(t, stagedPostURL, req.URL.String())
		require.Equal(t, http.MethodPost, req.Method)
		require.Empty(t, req.Header.Get("X-Shopify-Access-Token"))
		require.Empty(t, req.Header.Get("Authorization"))
		require.Nil(t, req.GetBody)
		reader, err := req.MultipartReader()
		require.NoError(t, err)
		for _, expected := range target.Parameters {
			part, err := reader.NextPart()
			require.NoError(t, err)
			require.Equal(t, expected.Name, part.FormName())
			value, err := io.ReadAll(part)
			require.NoError(t, err)
			require.Equal(t, expected.Value, string(value))
		}
		file, err := reader.NextPart()
		require.NoError(t, err)
		require.Equal(t, "file", file.FormName())
		require.Equal(t, "image-1.png", file.FileName())
		require.Equal(t, stagedMimeType, file.Header.Get("Content-Type"))
		data, err := io.ReadAll(file)
		require.NoError(t, err)
		require.Equal(t, contents, data)
		_, err = reader.NextPart()
		require.ErrorIs(t, err, io.EOF)
		return fixtureResponse(http.StatusCreated, ""), nil
	})}
	require.NoError(t, target.upload(context.Background(), client, "image-1.png", contents))

	draft := productDraft()
	draft.Images = []DraftImage{{URL: target.ResourceURL}}
	draft.Variants[0].ImageIndex = 1
	input, err := draft.input()
	require.NoError(t, err)
	require.Equal(t, []draftFile{{URL: target.ResourceURL, Filename: "image-1.png", ContentType: "IMAGE"}}, input.Files)
	require.Equal(t, &input.Files[0], input.Variants[0].File)
}

func TestStageRejectsNondevelopmentWriterAndOversizeBeforeRequest(t *testing.T) {
	client := &http.Client{Transport: transportFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("an invalid staging request reached Shopify")
		return nil, nil
	})}
	w := &DraftWrite{client: NewAdminClient(client), expiresAt: time.Now().Add(time.Minute)}
	_, err := w.StageImages(context.Background(), [][]byte{[]byte("\x89PNG\r\n\x1a\nfixture")})
	require.ErrorIs(t, err, ErrInvalidRequest)
	w.client = NewDevelopmentAdminClient(client)
	_, err = w.StageImages(context.Background(), [][]byte{make([]byte, MaxStagedDraftImageBytes+1)})
	require.ErrorIs(t, err, ErrInvalidRequest)
}
