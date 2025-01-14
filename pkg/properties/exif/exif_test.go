package exif

import (
	"context"
	"os"
	"testing"
)

func TestExifProcessor(t *testing.T) {
	t.Parallel()

	p := ExifProcessor{}

	bs, err := os.ReadFile("fixtures/exif.json")
	if err != nil {
		t.Fatalf("Could not read fixtures: %s", err)
	}

	props, err := p.Process(context.Background(), bs)
	if err != nil {
		t.Fatalf("Could not process exif: %s", err)
	}

	if got, exp := len(props), 14; got != exp {
		t.Fatalf("Expected %d properties, got %d", exp, got)
	}

	p0 := props[0]

	if got, exp := p0.PropertySource, "exif"; got != exp {
		t.Fatalf("Expected PropertySource to be %s, got %s", exp, got)
	}

	if got, exp := p0.PropertyType, "ApertureValue"; got != exp {
		t.Fatalf("Expected PropertyType to be %s, got %s", exp, got)
	}
}
