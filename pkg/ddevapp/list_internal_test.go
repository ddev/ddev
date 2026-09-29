package ddevapp

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFitColumnWidths(t *testing.T) {
	tests := []struct {
		name      string
		natural   []int
		available int
		want      []int
	}{
		{"everything fits", []int{12, 7, 30, 22, 8}, 100, []int{12, 7, 30, 22, 8}},
		{"widest columns capped evenly", []int{45, 7, 69, 22, 8}, 107, []int{35, 7, 35, 22, 8}},
		{"short columns keep their width", []int{20, 7, 69, 22, 8}, 83, []int{20, 7, 26, 22, 8}},
		{"all wide columns shrink together", []int{45, 7, 69, 22, 8}, 63, []int{16, 7, 16, 16, 8}},
		{"minimum width when nothing fits", []int{45, 7, 69, 40, 8}, 20, []int{7, 7, 7, 7, 7}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, fitColumnWidths(tc.natural, tc.available))
		})
	}
}
