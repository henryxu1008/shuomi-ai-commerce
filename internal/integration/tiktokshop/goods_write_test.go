package tiktokshop

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	marketplace "task-processor/internal/marketplace/tiktokshop"
)

func TestGlobalCreateSignsExactJSONAndNeverRetriesLostResponse(t *testing.T) {
	calls := 0
	client := newFixtureClient(t, func(request *http.Request) (*http.Response, error) {
		calls++
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Fatal(err)
		}
		if request.Method != "POST" || request.URL.Path != "/product/202309/global_products" || request.URL.Query().Has("shop_cipher") || request.Header.Get("x-tts-access-token") != "private-token" {
			t.Fatal("wrong global request")
		}
		expected, err := signature(request.URL.Path, request.URL.Query(), body, request.Header.Get("Content-Type"), clientSecretForTest)
		if err != nil {
			t.Fatal(err)
		}
		if request.URL.Query().Get("sign") != expected {
			t.Fatal("body changed after signing")
		}
		return nil, io.ErrUnexpectedEOF
	})
	input := marketplace.CreateGlobalRequest{ProductContent: marketplace.ProductContent{Title: "Title", Description: "<p>Text</p>", CategoryID: "cat", CategoryVersion: "v2"}, SKUs: []marketplace.GlobalSKU{{SellerSKU: "SKU"}}, ExternalID: "original-product"}
	_, err := client.CreateGlobalProduct(context.Background(), "private-token", input)
	if err != marketplace.ErrPublicationUnknown || calls != 1 {
		t.Fatalf("lost response: %v calls=%d", err, calls)
	}
}

const clientSecretForTest = "fixture-secret-never-log"

func TestImageUploadUsesMultipartDataAndExplicitUseCase(t *testing.T) {
	client := newFixtureClient(t, func(request *http.Request) (*http.Response, error) {
		if request.URL.Path != "/product/202309/images/upload" {
			t.Fatal("wrong image path")
		}
		if err := request.ParseMultipartForm(12 << 20); err != nil {
			t.Fatal(err)
		}
		defer request.MultipartForm.RemoveAll()
		file, _, err := request.FormFile("data")
		if err != nil {
			t.Fatal(err)
		}
		defer file.Close()
		content, _ := io.ReadAll(file)
		if string(content) != "approved-bytes" || request.FormValue("use_case") != "MAIN_IMAGE" {
			t.Fatal("image input changed")
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"code":0,"data":{"uri":"tos/image","url":"https://image.example.org/image.png","width":600,"height":600,"use_case":"MAIN_IMAGE"}}`))}, nil
	})
	image, err := client.UploadProductImage(context.Background(), "private-token", "MAIN_IMAGE", "image/png", []byte("approved-bytes"))
	if err != nil || image.URI != "tos/image" {
		t.Fatalf("image upload: %v", err)
	}
}
