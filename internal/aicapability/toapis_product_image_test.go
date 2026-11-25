package aicapability

import "testing"

func TestToAPIsReferenceIsAnExactPublicProductImage(t *testing.T) {
	const prefix = "https://cbu01.alicdn.com/img/ibank/"
	if !ToAPIsProductImageSource(prefix + "O1CN01qxjgGJ1FRGNAgQvI3_!!2219514910483-0-cib.jpg") {
		t.Fatal("expected the original 1688 product image path to be accepted")
	}
	for _, raw := range []string{
		"https://example.com/private/coat.jpg",
		"https://key@cbu01.alicdn.com/img/ibank/coat.jpg",
		"https://cbu01.alicdn.com:443/img/ibank/coat.jpg",
		prefix + "coat.jpg?access=secret",
		prefix + "coat.jpg?",
		prefix + "coat.jpg#fragment",
		prefix + "../ibank/coat.jpg",
		prefix + "./coat.jpg",
		prefix + "%2E/coat.jpg",
		prefix + "private/coat.jpg",
		prefix + "coat.jpg%00",
		prefix + "coat.webp",
	} {
		if ToAPIsProductImageSource(raw) {
			t.Errorf("accepted disallowed reference %q", raw)
		}
	}
}
