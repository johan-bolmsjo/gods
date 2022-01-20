package iter_test

import (
	"fmt"
	"testing"

	"github.com/johan-bolmsjo/gods/v2/iter"
)

type SimpleIterator []int

func (iter *SimpleIterator) Next() (int, bool) {
	if len(*iter) > 0 {
		v := (*iter)[0]
		(*iter) = (*iter)[1:]
		return v, true
	}
	return 0, false
}

func TestIterator(t *testing.T) {
	simpleIter := SimpleIterator{1, 2, 3}
	var output []int

	scanner := iter.NewScanner[int](&simpleIter)
	for scanner.Scan() {
		output = append(output, scanner.Result())
	}

	if got, want := fmt.Sprint(output), "[1 2 3]"; got != want {
		t.Fatalf("got sequence %v; want %v", got, want)
	}
}

type SimplePair struct {
	key   int
	value string
}
type SimplePairIterator []SimplePair

func (iter *SimplePairIterator) Next() (int, string, bool) {
	if len(*iter) > 0 {
		k, v := (*iter)[0].key, (*iter)[0].value
		(*iter) = (*iter)[1:]
		return k, v, true
	}
	return 0, "", false
}

func TestPairIterator(t *testing.T) {
	simplePairIter := SimplePairIterator{{1, "banana"}, {2, "apple"}, {3, "lemon"}}
	var output []SimplePair

	scanner := iter.NewPairScanner[int, string](&simplePairIter)
	for scanner.Scan() {
		k, v := scanner.Result()
		output = append(output, SimplePair{k, v})
	}

	if got, want := fmt.Sprint(output), "[{1 banana} {2 apple} {3 lemon}]"; got != want {
		t.Fatalf("got sequence %v; want %v", got, want)
	}
}
