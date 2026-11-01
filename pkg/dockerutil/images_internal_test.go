package dockerutil

import (
	"testing"

	"github.com/moby/moby/api/types/image"
	"github.com/stretchr/testify/require"
)

// TestImageMatches covers the check FindImagesByLabels applies to what the
// API returns, for an API that ignores the label or dangling filters.
func TestImageMatches(t *testing.T) {
	labels := map[string]string{"com.ddev.site-name": "proj", "com.ddev.buildhost": ""}
	tests := []struct {
		name         string
		img          image.Summary
		danglingOnly bool
		want         bool
	}{
		{"all labels", image.Summary{Labels: map[string]string{"com.ddev.site-name": "proj", "com.ddev.buildhost": "h"}, RepoTags: []string{"x:y"}}, false, true},
		{"wrong value", image.Summary{Labels: map[string]string{"com.ddev.site-name": "other", "com.ddev.buildhost": "h"}}, false, false},
		{"missing key", image.Summary{Labels: map[string]string{"com.ddev.site-name": "proj"}}, false, false},
		{"no labels", image.Summary{RepoTags: []string{"busybox:latest"}}, false, false},
		{"dangling", image.Summary{Labels: map[string]string{"com.ddev.site-name": "proj", "com.ddev.buildhost": "h"}, RepoTags: []string{"<none>:<none>"}}, true, true},
		{"untagged", image.Summary{Labels: map[string]string{"com.ddev.site-name": "proj", "com.ddev.buildhost": "h"}}, true, true},
		{"tagged, dangling only", image.Summary{Labels: map[string]string{"com.ddev.site-name": "proj", "com.ddev.buildhost": "h"}, RepoTags: []string{"x:y"}}, true, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, imageMatches(tc.img, labels, tc.danglingOnly))
		})
	}
}
