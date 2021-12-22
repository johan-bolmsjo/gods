package math_test

import (
	"testing"

	"github.com/johan-bolmsjo/gods/v2/math"
)

func TestCompareOrdered(t *testing.T) {
	testData := [][3]int{
		{-100, 100, -1},
		{100, -100, 1},
		{0, 0, 0},
	}
	for _, td := range testData {
		if got, want := math.CompareOrdered(td[0], td[1]), td[2]; got != want {
			t.Fatalf("math.CompareOrdered(%d, %d) = %d; want %d", td[0], td[1], got, want)
		}
	}
}

func TestAbsSigned(t *testing.T) {
	testData := [][2]int{
		{-100, 100},
		{100, 100},
	}
	for _, td := range testData {
		if got, want := math.AbsSigned(td[0]), td[1]; got != want {
			t.Fatalf("math.AbsSigned(%d) = %d; want %d", td[0], got, want)
		}
	}
}

func TestMinInteger(t *testing.T) {
	testData := [][3]int{
		{-100, 100, -100},
		{100, -100, -100},
		{100, 100, 100},
	}
	for _, td := range testData {
		if got, want := math.MinInteger(td[0], td[1]), td[2]; got != want {
			t.Fatalf("math.MinInteger(%d, %d) = %d; want %d", td[0], td[1], got, want)
		}
	}
}

func TestMaxInteger(t *testing.T) {
	testData := [][3]int{
		{-100, 100, 100},
		{100, -100, 100},
		{100, 100, 100},
	}
	for _, td := range testData {
		if got, want := math.MaxInteger(td[0], td[1]), td[2]; got != want {
			t.Fatalf("math.MaxInteger(%d, %d) = %d; want %d", td[0], td[1], got, want)
		}
	}
}
