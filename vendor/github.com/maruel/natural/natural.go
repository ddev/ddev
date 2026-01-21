// Copyright 2018 Marc-Antoine Ruel. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

// Package natural defines a natural "less" to compare two strings while
// interpreting natural numbers.
//
// This is occasionally nicknamed 'natsort'.
//
// It does so with no memory allocation.
package natural

import (
	"strconv"
	"strings"
)

// Less does a 'natural' comparison on the two strings.
//
// It treats digits as decimal numbers, so that Less("10", "2") return false.
//
// This function does no memory allocation.
func Less(a, b string) bool {
	return Compare(a, b) < 0
}

// Compare does a 'natural' comparison on the two strings.
//
// It treats digits as decimal numbers, so that Compare("10", "2") return >0.
//
// This function does no memory allocation.
func Compare(a, b string) int {
	for {
		if p := commonPrefix(a, b); p != 0 {
			a = a[p:]
			b = b[p:]
		}
		if len(a) == 0 {
			return -len(b)
		}
		if ia := digits(a); ia > 0 {
			if ib := digits(b); ib > 0 {
				// Both sides have digits.
				an, aerr := strconv.ParseUint(a[:ia], 10, 64)
				bn, berr := strconv.ParseUint(b[:ib], 10, 64)
				if aerr == nil && berr == nil {
					// Fast path: both fit in uint64
					if an != bn {
						// #nosec G40
						return int(an - bn)
					}
					// Semantically the same digits, e.g. "00" == "0", "01" == "1". In
					// this case, only continue processing if there's trailing data on
					// both sides, otherwise do lexical comparison.
					if ia != len(a) && ib != len(b) {
						a = a[ia:]
						b = b[ib:]
						continue
					}
				} else {
					// Slow path: at least one number exceeds uint64
					// Both are still pure digits (verified by ia > 0 and ib > 0)
					result := compareNumericStrings(a[:ia], b[:ib])
					if result != 0 {
						return result
					}
					// Numbers are semantically equal, continue if both have trailing data
					if ia != len(a) && ib != len(b) {
						a = a[ia:]
						b = b[ib:]
						continue
					}
				}
			}
		}
		return strings.Compare(a, b)
	}
}

// StringSlice attaches the methods of Interface to []string, sorting in
// increasing order using natural order.
//
// It is now obsolete, use slices.Sort() along with natural.Compare instead.
type StringSlice []string

func (p StringSlice) Len() int           { return len(p) }
func (p StringSlice) Less(i, j int) bool { return Less(p[i], p[j]) }
func (p StringSlice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

//

// commonPrefix returns the common prefix except for digits.
func commonPrefix(a, b string) int {
	m := len(a)
	if n := len(b); n < m {
		m = n
	}
	if m == 0 {
		return 0
	}
	_ = a[m-1]
	_ = b[m-1]
	for i := 0; i < m; i++ {
		ca := a[i]
		cb := b[i]
		if (ca >= '0' && ca <= '9') || (cb >= '0' && cb <= '9') || ca != cb {
			return i
		}
	}
	return m
}

func digits(s string) int {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			return i
		}
	}
	return len(s)
}

// compareNumericStrings compares two numeric strings without parsing them.
// This handles arbitrarily large numbers that don't fit in uint64.
// It does no memory allocation.
func compareNumericStrings(a, b string) int {
	// Strip leading zeros
	a = trimLeadingZeros(a)
	b = trimLeadingZeros(b)

	// Compare by length first (more digits = larger number)
	if len(a) != len(b) {
		return len(a) - len(b)
	}

	// Same length: lexical comparison works correctly for digits
	return strings.Compare(a, b)
}

// trimLeadingZeros removes leading zeros from a numeric string.
// Returns "0" if the string is all zeros.
// This function does no memory allocation (only string slicing).
func trimLeadingZeros(s string) string {
	for i := 0; i < len(s); i++ {
		if s[i] != '0' {
			return s[i:]
		}
	}
	// All zeros - return "0"
	if len(s) > 0 {
		return s[len(s)-1:]
	}
	return s
}
