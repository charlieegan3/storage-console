package browse

import (
	"testing"
)

func TestBreadcrumbsFromPath(t *testing.T) {
	tests := map[string]struct {
		inputPath string
		expected  breadcrumbs
	}{
		"root case": {
			inputPath: "",
			expected: breadcrumbs{
				Display: false,
				Items:   []breadcrumb{},
			},
		},
		"single level": {
			inputPath: "a",
			expected: breadcrumbs{
				Display: true,
				Items: []breadcrumb{
					{
						Name:      "root",
						Path:      "/",
						Navigable: true,
					},
					{
						Name:      "a",
						Path:      "/a",
						Navigable: false,
					},
				},
			},
		},
		"single level with a trailing slash": {
			inputPath: "a/",
			expected: breadcrumbs{
				Display: true,
				Items: []breadcrumb{
					{
						Name:      "root",
						Path:      "/",
						Navigable: true,
					},
					{
						Name:      "a",
						Path:      "/a",
						Navigable: false,
					},
				},
			},
		},
		"single level with a trailing slash and a leaning one": {
			inputPath: "/a/",
			expected: breadcrumbs{
				Display: true,
				Items: []breadcrumb{
					{
						Name:      "root",
						Path:      "/",
						Navigable: true,
					},
					{
						Name:      "a",
						Path:      "/a",
						Navigable: false,
					},
				},
			},
		},
		"three levels": {
			inputPath: "/a/b/c",
			expected: breadcrumbs{
				Display: true,
				Items: []breadcrumb{
					{
						Name:      "root",
						Path:      "/",
						Navigable: true,
					},
					{
						Name:      "a",
						Path:      "/a",
						Navigable: true,
					},
					{
						Name:      "b",
						Path:      "/a/b",
						Navigable: true,
					},
					{
						Name:      "c",
						Path:      "/a/b/c",
						Navigable: false,
					},
				},
			},
		},
		"three levels with file": {
			inputPath: "/a/b/c/file.jpg",
			expected: breadcrumbs{
				Display: true,
				Items: []breadcrumb{
					{
						Name:      "root",
						Path:      "/",
						Navigable: true,
					},
					{
						Name:      "a",
						Path:      "/a",
						Navigable: true,
					},
					{
						Name:      "b",
						Path:      "/a/b",
						Navigable: true,
					},
					{
						Name:      "c",
						Path:      "/a/b/c",
						Navigable: true,
					},
					{
						Name:      "file.jpg",
						Path:      "/a/b/c/file.jpg",
						Navigable: false,
					},
				},
			},
		},
	}

	for testCase, testData := range tests {
		t.Run(testCase, func(t *testing.T) {
			actual := breadcrumbsFromPath(testData.inputPath)

			if actual.Display != testData.expected.Display {
				t.Errorf("expected %v, got %v", testData.expected.Display, actual.Display)
			}

			if len(actual.Items) != len(testData.expected.Items) {
				t.Fatalf("expected %v, got %v", len(testData.expected.Items), len(actual.Items))
			}

			for i, item := range actual.Items {
				if item.Name != testData.expected.Items[i].Name {
					t.Errorf("expected %v, got %v", testData.expected.Items[i].Name, item.Name)
				}
				if item.Path != testData.expected.Items[i].Path {
					t.Errorf("expected %v, got %v", testData.expected.Items[i].Path, item.Path)
				}
				if item.Navigable != testData.expected.Items[i].Navigable {
					t.Errorf("expected %v, got %v", testData.expected.Items[i].Navigable, item.Navigable)
				}
			}
		})
	}
}
