package util

import (
	"fmt"
	"testing"

	asrt "github.com/stretchr/testify/assert"
)

// TestRandString ensures that RandString only generates string of the correct value and characters.
func TestRandString(t *testing.T) {
	assert := asrt.New(t)
	stringLengths := []int{2, 4, 8, 16, 23, 47}

	for _, stringLength := range stringLengths {
		testString := RandString(stringLength)
		assert.Equal(len(testString), stringLength, fmt.Sprintf("Generated string is of length %d", stringLengths))
	}

	lb := "a"
	setLetterBytes(lb)
	testString := RandString(1)
	assert.Equal(testString, lb)
}
