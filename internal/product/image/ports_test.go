package image

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCanonicalHTTPURLNormalizesDNSAndIPIdentities(t *testing.T) {
	t.Parallel()

	for name, testCase := range map[string]struct {
		raw  string
		want string
	}{
		"absolute DNS name, casing, non-default port, path and query": {
			raw:  "HTTPS://SOURCE.EXAMPLE.:8443/catalog/../a.png?z=2&a=1",
			want: "https://source.example:8443/a.png?a=1&z=2",
		},
		"IPv6 literal and default port": {
			raw:  "https://[2001:0DB8:0:0:0:0:0:1]:443/catalog/../a.png?z=2&a=1",
			want: "https://[2001:db8::1]/a.png?a=1&z=2",
		},
		"IPv6 literal and non-default port": {
			raw:  "http://[2001:DB8::1]:8080/a.png",
			want: "http://[2001:db8::1]:8080/a.png",
		},
	} {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, testCase.want, canonicalHTTPURL(testCase.raw))
		})
	}
}
func TestCanonicalHTTPURLNormalizesStrictExplicitPorts(t *testing.T) {
	t.Parallel()

	for name, testCase := range map[string]struct {
		raw  string
		want string
	}{
		"HTTPS numeric default with leading zeros": {
			raw: "https://source.example:00443/a.png", want: "https://source.example/a.png",
		},
		"HTTP numeric default with leading zeros": {
			raw: "http://source.example:00080/a.png", want: "http://source.example/a.png",
		},
		"DNS non-default with leading zero": {
			raw: "https://source.example:08443/a.png", want: "https://source.example:8443/a.png",
		},
		"IPv6 numeric default with leading zeros": {
			raw: "https://[2001:DB8::1]:00443/a.png", want: "https://[2001:db8::1]/a.png",
		},
		"IPv6 non-default with leading zero": {
			raw: "https://[2001:DB8::1]:08443/a.png", want: "https://[2001:db8::1]:8443/a.png",
		},
		"lowest valid port": {
			raw: "https://source.example:1/a.png", want: "https://source.example:1/a.png",
		},
		"highest valid port": {
			raw: "https://source.example:65535/a.png", want: "https://source.example:65535/a.png",
		},
	} {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, testCase.want, canonicalHTTPURL(testCase.raw))
		})
	}
}
func TestCanonicalHTTPURLRejectsInvalidExplicitPortsAndAuthority(t *testing.T) {
	t.Parallel()

	for name, raw := range map[string]string{
		"DNS trailing colon":       "https://source.example:/a.png",
		"IPv6 trailing colon":      "https://[2001:db8::1]:/a.png",
		"non-numeric port":         "https://source.example:abc/a.png",
		"signed port":              "https://source.example:-443/a.png",
		"explicit zero port":       "https://source.example:000/a.png",
		"port above 65535":         "https://source.example:65536/a.png",
		"userinfo":                 "https://user@source.example/a.png",
		"fragment":                 "https://source.example/a.png#fragment",
		"invalid escaped hostname": "https://source%2Fexample.com/a.png",
	} {
		t.Run(name, func(t *testing.T) {
			require.Empty(t, canonicalHTTPURL(raw))
		})
	}
}
func TestCanonicalHTTPURLRejectsMalformedRawAuthoritiesAcrossURLStrictModes(t *testing.T) {
	for _, mode := range []struct {
		name    string
		godebug string
	}{
		{name: "default"},
		{name: "urlstrictcolons disabled", godebug: "urlstrictcolons=0"},
	} {
		t.Run(mode.name, func(t *testing.T) {
			t.Setenv("GODEBUG", mode.godebug)
			for name, raw := range map[string]string{
				"empty authority":            "https:///a.png",
				"userinfo":                   "https://user@source.example/a.png",
				"DNS authority with colons":  "https://source.example:80:90/a.png",
				"unbracketed IPv6 authority": "https://2001:db8::1/a.png",
				"stray opening bracket":      "https://source[.example/a.png",
				"extra closing bracket":      "https://[2001:db8::1]]/a.png",
				"bracketed suffix colon":     "https://[2001:db8::1]:443:90/a.png",
			} {
				t.Run(name, func(t *testing.T) {
					require.Empty(t, canonicalHTTPURL(raw))
				})
			}
		})
	}
}
func TestCanonicalHTTPURLAcceptsValidRawAuthoritiesAcrossURLStrictModes(t *testing.T) {
	for _, mode := range []struct {
		name    string
		godebug string
	}{
		{name: "default"},
		{name: "urlstrictcolons disabled", godebug: "urlstrictcolons=0"},
	} {
		t.Run(mode.name, func(t *testing.T) {
			t.Setenv("GODEBUG", mode.godebug)
			for name, testCase := range map[string]struct {
				raw  string
				want string
			}{
				"DNS and port": {
					raw: "https://source.example:8443/a.png", want: "https://source.example:8443/a.png",
				},
				"bracketed IPv6": {
					raw: "https://[2001:0DB8::1]:443/a.png", want: "https://[2001:db8::1]/a.png",
				},
				"bracketed IPv6 zone": {
					raw: "http://[fe80::1%25eth0]:8080/a.png", want: "http://[fe80::1%25eth0]:8080/a.png",
				},
				"path and query": {
					raw: "https://source.example/catalog/../a.png?z=2&a=1", want: "https://source.example/a.png?a=1&z=2",
				},
			} {
				t.Run(name, func(t *testing.T) {
					require.Equal(t, testCase.want, canonicalHTTPURL(testCase.raw))
				})
			}
		})
	}
}
func TestCanonicalHTTPURLFailsClosedOnMalformedQuery(t *testing.T) {
	t.Parallel()

	for name, raw := range map[string]string{
		"semicolon separator": "https://source.example/a.png?first=1;second=2",
		"bad query escape":    "https://source.example/a.png?first=%zz",
	} {
		t.Run(name, func(t *testing.T) {
			require.Empty(t, canonicalHTTPURL(raw))
		})
	}
}
func TestCanonicalHTTPURLDefinesRepeatedQueryKeyOrdering(t *testing.T) {
	t.Parallel()

	first := canonicalHTTPURL("https://source.example/a.png?z=9&a=2&a=1&b=3")
	second := canonicalHTTPURL("https://source.example/a.png?b=3&a=2&a=1&z=9")
	reversedValues := canonicalHTTPURL("https://source.example/a.png?b=3&a=1&a=2&z=9")

	require.Equal(t, "https://source.example/a.png?a=2&a=1&b=3&z=9", first)
	require.Equal(t, first, second, "query key order is not part of canonical identity")
	require.NotEqual(t, first, reversedValues, "repeated values preserve their order")
}
func TestCanonicalHTTPURLRejectsOversizedRawInputBeforeParsing(t *testing.T) {
	const designMaxImageStringBytes = 8 << 10
	oversized := "https://source.example/" + strings.Repeat("x", designMaxImageStringBytes)
	require.Greater(t, len(oversized), designMaxImageStringBytes)

	allocations := testing.AllocsPerRun(100, func() {
		canonicalURLSink = canonicalHTTPURL(oversized)
	})
	require.Empty(t, canonicalURLSink)
	require.Zero(t, allocations, "oversized raw URLs must return before URL parsing allocates")
}
func TestStringCollectionValidatorsPreflightBeforeCloneAllocation(t *testing.T) {
	const designMaxImageStringBytes = 8 << 10
	overlong := strings.Repeat("x", designMaxImageStringBytes+1)

	t.Run("product attributes", func(t *testing.T) {
		input := ProductContext{ProductKey: "product-1", Attributes: map[string]string{overlong: "value"}}
		allocations := testing.AllocsPerRun(10, func() {
			productContextSink, preflightErrorSink = validateProductContext(input)
		})
		require.ErrorIs(t, preflightErrorSink, ErrInputInvalid)
		require.Equal(t, ProductContext{}, productContextSink)
		require.Zero(t, allocations)
	})

	t.Run("operations", func(t *testing.T) {
		allocations := testing.AllocsPerRun(10, func() {
			stringSliceSink, preflightErrorSink = validateOperations([]string{overlong}, false)
		})
		require.ErrorIs(t, preflightErrorSink, ErrInputInvalid)
		require.Nil(t, stringSliceSink)
		require.Zero(t, allocations)
	})

	t.Run("normalized strings", func(t *testing.T) {
		allocations := testing.AllocsPerRun(10, func() {
			stringSliceSink, preflightErrorSink = normalizedStrings([]string{overlong}, 1)
		})
		require.ErrorIs(t, preflightErrorSink, ErrInputInvalid)
		require.Nil(t, stringSliceSink)
		require.Zero(t, allocations)
	})
}
func TestProductContextRawResourcePreflightIsAllocationFreeAndCanonicalAgnostic(t *testing.T) {
	boundedNonCanonical := ProductContext{
		ProductKey: " product-1 ",
		Attributes: map[string]string{" material ": " steel "},
	}
	allocations := testing.AllocsPerRun(10, func() {
		preflightErrorSink = preflightProductContextResources(boundedNonCanonical)
	})
	require.NoError(t, preflightErrorSink)
	require.Zero(t, allocations, "raw resource preflight must not trim, canonicalize or clone")

	got, err := validateProductContext(boundedNonCanonical)
	require.ErrorIs(t, err, ErrInputInvalid)
	require.Equal(t, ProductContext{}, got)
}
func TestProductContextRejectsRawAggregateBeforeCanonicalPhase(t *testing.T) {
	const designMaxImageStringBytes = 8 << 10
	attributes := make(map[string]string, 8)
	for index := 0; index < 8; index++ {
		attributes[string(rune('a'+index))] = strings.Repeat(string(rune('a'+index)), designMaxImageStringBytes)
	}
	input := ProductContext{ProductKey: " product-1 ", Attributes: attributes}

	allocations := testing.AllocsPerRun(10, func() {
		preflightErrorSink = preflightProductContextResources(input)
	})
	require.ErrorIs(t, preflightErrorSink, ErrInputInvalid)
	require.Zero(t, allocations, "aggregate overflow must stop before canonical checks or map cloning")
}
func TestGeneratedArtifactAggregatePreflightUsesOverflowSafeBudget(t *testing.T) {
	t.Parallel()

	inline := func(content string) Candidate {
		return Candidate{Asset: validInlineGeneratedAsset(RoleScene, "render_scene", []byte(content))}
	}
	remote := Candidate{Asset: validGeneratedAsset(RoleScene, "render_scene", "https://cdn.example/remote.png")}
	for name, testCase := range map[string]struct {
		candidates []Candidate
		wantErr    bool
	}{
		"exact small budget": {candidates: []Candidate{inline("1234"), inline("5678")}},
		"one byte over":      {candidates: []Candidate{inline("1234"), inline("5678"), inline("9")}, wantErr: true},
		"remote does not consume inline budget": {
			candidates: []Candidate{remote, inline("12345678")},
		},
	} {
		t.Run(name, func(t *testing.T) {
			err := validateGeneratedArtifactAggregate(testCase.candidates, 8)
			if testCase.wantErr {
				require.ErrorIs(t, err, ErrOutputValidation)
				return
			}
			require.NoError(t, err)
		})
	}
}
func validProductContext() ProductContext {
	return ProductContext{
		ProductKey:  "product-1",
		Title:       "Steel bottle",
		ProductType: "bottle",
		Attributes:  map[string]string{"material": "steel"},
	}
}
func validSourceAsset() Asset {
	return Asset{
		URL:           "https://source.example:443/a.png",
		SourceURL:     "https://origin.example/a.png",
		SourceAssetID: "source-1",
		Role:          RoleSource,
		Width:         1200,
		Height:        1200,
		Operations:    []string{"source"},
	}
}
func validGeneratedAsset(role Role, operation, imageURL string) Asset {
	return Asset{
		URL:           imageURL,
		MediaType:     "image/png",
		SourceURL:     "https://source.example:443/a.png",
		SourceAssetID: "source-1",
		Role:          role,
		Width:         1200,
		Height:        1200,
		Operations:    []string{operation},
	}
}
func validInlineGeneratedAsset(role Role, operation string, content []byte) Asset {
	return Asset{
		Bytes:         content,
		MediaType:     "image/png",
		SourceURL:     "https://source.example:443/a.png",
		SourceAssetID: "source-1",
		Role:          role,
		Width:         1200,
		Height:        1200,
		Operations:    []string{operation},
	}
}
func validUsageAuthorization(operation string) UsageQuote {
	return UsageQuote{
		Operation: operation, Provider: "provider-a", RouteReference: "route-a", Model: "model-a",
		CredentialReference: "credential-a", ConfigurationVersion: "config-v1", PricingVersion: "pricing-v1",
		Fingerprint: operation + "-quote-v1", MaximumOutputs: 1, MaximumModelCalls: 1,
		MaximumCostMicros: 10, CostUpperBoundKnown: true,
	}
}

var (
	productContextSink ProductContext
	stringSliceSink    []string
	preflightErrorSink error
	canonicalURLSink   string
)
