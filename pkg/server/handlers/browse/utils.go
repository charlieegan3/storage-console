package browse

import (
	"fmt"
	"math"
	"strings"
)

func humanizeBytes(bytes int64) string {
	suffixes := []string{"B", "KB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB"}

	base := 1024.0
	if bytes == 0 {
		return fmt.Sprintf("0%s", suffixes[0])
	}

	exp := math.Floor(math.Log(float64(bytes)) / math.Log(base))
	index := int(math.Min(exp, float64(len(suffixes)-1)))
	value := float64(bytes) / math.Pow(base, exp)

	if value > 10 {
		return fmt.Sprintf("%.0f%s", value, suffixes[index])
	}

	return fmt.Sprintf("%.1f%s", value, suffixes[index])
}

func breadcrumbsFromPath(path string) breadcrumbs {

	// root case where there are no breadcrumbs to show
	if path == "" {
		return breadcrumbs{
			Display: false,
			Items:   []breadcrumb{},
		}
	}

	b := breadcrumbs{
		Display: true,
		Items: []breadcrumb{
			{
				Name:      "root",
				Path:      "/",
				Navigable: true,
			},
		},
	}

	path = strings.TrimSuffix(path, "/")
	path = strings.TrimPrefix(path, "/")

	parts := strings.Split(path, "/")

	for i, part := range parts {
		if part == "" {
			continue
		}

		bcPath := "/" + strings.Join(parts[:i+1], "/")

		b.Items = append(b.Items, breadcrumb{
			Name:      part,
			Path:      bcPath,
			Navigable: len(parts)-1 != i,
		})
	}

	return b
}
