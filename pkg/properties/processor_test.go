package properties

import (
	"testing"
	"time"
)

func TestBlobPropertiesColor(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		BlobProperties BlobProperties
		Color          string
	}{
		"predef color": {
			BlobProperties: BlobProperties{
				PropertySource: "color",
				PropertyType:   "ColorCategory1",
				ValueType:      "Text",
				ValueText:      &[]string{"ro"}[0],
			},
			Color: "255,127,0",
		},
		"color": {
			BlobProperties: BlobProperties{
				PropertySource: "color",
				PropertyType:   "ProminentColor1",
				ValueType:      "Text",
				ValueText:      &[]string{"1,2,3"}[0],
			},
			Color: "1,2,3",
		},
	}

	for tcName, testData := range testCases {
		t.Run(tcName, func(t *testing.T) {
			t.Parallel()

			if got, exp := testData.BlobProperties.Color(), testData.Color; got != exp {
				t.Fatalf("expected: %q, got %q", exp, got)
			}
		})
	}
}

func TestBlobPropertiesString(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		BlobProperties BlobProperties
		String         string
	}{
		"bool": {
			BlobProperties: BlobProperties{
				ValueType: "Bool",
				ValueBool: &[]bool{true}[0],
			},
			String: "True",
		},
		"text": {
			BlobProperties: BlobProperties{
				ValueType: "Text",
				ValueText: &[]string{"wow"}[0],
			},
			String: "wow",
		},
		"fraction": {
			BlobProperties: BlobProperties{
				ValueType:        "Fraction",
				ValueNumerator:   &[]int{10}[0],
				ValueDenominator: &[]int{100}[0],
			},
			String: "10/100",
		},
		"int": {
			BlobProperties: BlobProperties{
				ValueType:    "Integer",
				ValueInteger: &[]int{10}[0],
			},
			String: "10",
		},
		"float": {
			BlobProperties: BlobProperties{
				ValueType:  "Float",
				ValueFloat: &[]float64{1.23}[0],
			},
			String: "1.23",
		},
		"timestamp": {
			BlobProperties: BlobProperties{
				ValueType:      "Timestamp",
				ValueTimestamp: &[]time.Time{time.Date(1996, time.June, 2, 5, 4, 3, 0, time.UTC)}[0],
			},
			String: "1996-06-02 05:04:03 +0000 UTC",
		},
	}

	for tcName, testData := range testCases {
		t.Run(tcName, func(t *testing.T) {
			t.Parallel()

			if got, exp := testData.BlobProperties.String(), testData.String; got != exp {
				t.Fatalf("expected: %q, got %q", exp, got)
			}
		})
	}
}
