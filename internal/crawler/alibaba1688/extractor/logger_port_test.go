package extractor

import (
	"io"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestProductExtractorPassesCurrentLoggerToEveryChild(t *testing.T) {
	first, second := logrus.New(), logrus.New()
	first.SetOutput(io.Discard)
	second.SetOutput(io.Discard)
	current := first
	logs := func(component string) *logrus.Entry {
		return current.WithFields(logrus.Fields{"component": component, "request_id": "synthetic-request"})
	}
	p := NewProductExtractor(logs)
	children := []interface{ componentLogger(string) *logrus.Entry }{
		p, p.basicInfoExtractor, p.categoryExtractor, p.shippingExtractor, p.variantExtractor,
		p.attributeExtractor, p.detailExtractor, p.packInfoExtractor, p.variantValuesExtractor,
		p.priceRangeExtractor, p.titleExtractor, p.priceExtractor, p.supplierExtractor,
		p.imageExtractor, p.specificationExtractor,
	}
	for _, raw := range []*logrus.Logger{first, second} {
		current = raw
		for _, child := range children {
			entry := child.componentLogger("crawler/alibaba1688")
			if entry.Logger != raw || entry.Data["component"] != "crawler/alibaba1688" || entry.Data["request_id"] != "synthetic-request" {
				t.Fatal("nested extractor lost the current factory, component or context")
			}
		}
	}
}
