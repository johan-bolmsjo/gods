package math

import "golang.org/x/exp/constraints"

// Compare two items and return a value less than, equal to, or greater than
// zero if lhs is found, respectively, to be less than, to match, or be greater
// than rhs.
type Comparator[T any] func(lhs, rhs T) int

// CompareOrdered compares two values satisfying constraints.Ordered and return
// a value less than, equal to, or greater than zero if lhs is found,
// respectively, to be less than, to match, or be greater than rhs.
func CompareOrdered[T constraints.Ordered](lhs, rhs T) int {
	if lhs < rhs {
		return -1
	} else if lhs == rhs {
		return 0
	}
	return 1
}

// AbsSigned returns the absolute value of a signed integer value.
func AbsSigned[T constraints.Signed](val T) T {
	if val < 0 {
		return -val
	}
	return val
}
