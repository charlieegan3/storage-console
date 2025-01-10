package color

import (
	"context"
	"fmt"
	"os"
	"testing"
)

func TestColorProcessor(t *testing.T) {
	t.Parallel()
	p := ColorProcessor{}

	bs, err := os.ReadFile("fixtures/color.json")
	if err != nil {
		t.Fatalf("Could not read fixtures: %s", err)
	}

	props, err := p.Process(context.Background(), bs)
	if err != nil {
		t.Fatalf("Could not process color: %s", err)
	}

	if exp, got := 6, len(props); exp != got {
		t.Fatalf("Expected %d properties, got %d", exp, got)
	}

	fmt.Println(props)

	p0 := props[0]

	if p0.PropertySource != "color" {
		t.Fatalf("Expected property source to be color, got %s", p0.PropertySource)
	}
}
