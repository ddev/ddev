package pantheon

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestInt64Parsing ensures the jsonInt64 parsing is working as expected.
func TestInt64Parsing(t *testing.T) {
	assert := assert.New(t)
	input := `[1, 20, "314", " 418   "]`
	results := []int64{1, 20, 314, 418}
	var nums []jsonInt64
	err := json.Unmarshal([]byte(input), &nums)
	assert.NoError(err)

	for i, v := range results {
		assert.Equal(v, int64(nums[i]))
	}
}
