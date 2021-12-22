package avltree_test

import (
	"fmt"
	"testing"

	"github.com/johan-bolmsjo/gods/v2/avltree"
	"github.com/johan-bolmsjo/gods/v2/math"
)

type keyType int
type valType int
type treeType = avltree.Tree[keyType, valType]
type iterType = avltree.Iterator[keyType, valType]
type treeOptionType = avltree.TreeOption[keyType, valType]

type assoc struct {
	key keyType
	val valType
}

// Brute force test of tree rotations triggered by inserting elements.
// Tree invariants are validated after each operation.
func TestInvariantsPermuteInsert(t *testing.T) {
	tree := newTree(nil, avltree.WithSyncPool[keyType, valType]())
	src := someKeys{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	var dst someKeys
	alen := len(src)

	seq := 0
	for permute(&dst, &src, seq) {
		for j := 0; j < alen; j++ {
			key := dst[j]
			tree.Add(key, valType(key))
			if _, ok := tree.Find(key); !ok {
				t.Fatalf("Failed to add key=%v, index=%v, sequence=%v", key, j, seq)
			}
			balanced, sorted := tree.Validate()
			if !balanced || !sorted {
				t.Fatalf("Invalid tree invariant: balanced=%v, sorted=%v, equence=%v", balanced, sorted, dst)
			}
		}
		tree.Clear(nil)
		seq++
	}
	t.Logf("%d insert sequences tested", seq)
}

// Brute force test of tree rotations triggered by removing elements.
// Tree invariants are validated after each operation.
func TestInvariantsPermuteRemove(t *testing.T) {
	tree := newTree(nil, avltree.WithSyncPool[keyType, valType]())
	src := someKeys{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	var dst someKeys
	alen := len(src)

	seq := 0
	for permute(&dst, &src, seq) {
		for j := 0; j < alen; j++ {
			key := src[j]
			tree.Add(key, valType(key))
			if _, ok := tree.Find(key); !ok {
				t.Fatalf("Failed to add key=%v, index=%v, sequence=%v", key, j, seq)
			}
		}

		for j := 0; j < alen; j++ {
			key := dst[j]
			tree.Remove(dst[j])
			if _, ok := tree.Find(key); ok {
				t.Fatalf("Failed to remove key=%v, index=%v, sequence=%v", key, j, seq)
			}
			balanced, sorted := tree.Validate()
			if !balanced || !sorted {
				t.Fatalf("Invalid tree invariant: balanced=%v, sorted=%v, sequence=%v", balanced, sorted, dst)
			}
		}
		tree.Clear(nil)
		seq++
	}
	t.Logf("%d remove sequences tested", seq)
}

// Adding a key that already exist should overwrite the existing association.
func TestAddExisting(t *testing.T) {
	tree := newTree(nil)

	const (
		key  = keyType(1)
		val1 = valType(1)
		val2 = valType(2)
	)

	tree.Add(key, val1)
	v, ok := tree.Find(key)
	if !ok || v != val1 {
		t.Fatalf("tree.Find() = (%v, %v); want (%v, %v)", v, ok, val1, true)
	}

	tree.Add(key, val2)
	v, ok = tree.Find(key)
	if !ok || v != val2 {
		t.Fatalf("tree.Find() = (%v, %v); want (%v, %v)", v, ok, val2, true)
	}
}

// Iterators should be updated by by insert and remove operations.
func TestIteratorUpdate(t *testing.T) {
	testData := []struct {
		name           string
		modifier       func(t *treeType)
		baseSeq        []keyType
		expectedFwdSeq []keyType
		expectedRevSeq []keyType
	}{
		{
			"Insert",
			func(t *treeType) { bulkInsert(t, []keyType{2, 10}) },
			[]keyType{1, 3, 5, 7, 9, 11},
			[]keyType{3, 5, 7, 9, 10, 11},
			[]keyType{9, 7, 5, 3, 2, 1},
		},
		{
			"Remove",
			func(t *treeType) { bulkRemove(t, []keyType{1, 11}) },
			[]keyType{1, 3, 5, 7, 9, 11},
			[]keyType{3, 5, 7, 9},
			[]keyType{9, 7, 5, 3},
		},
		{
			"RemoveAndMove", // Iterator should be moved by remove operation
			func(t *treeType) { bulkRemove(t, []keyType{3, 9}) },
			[]keyType{1, 3, 5, 7, 9, 11},
			[]keyType{5, 7, 11},
			[]keyType{7, 5, 1},
		},
		{
			"RemoveAndInvalidate", // Iterator should fall off the edge and be invalidated
			func(t *treeType) { bulkRemove(t, []keyType{1, 3, 5}) },
			[]keyType{1, 3, 5},
			[]keyType{},
			[]keyType{},
		},
	}

	for _, td := range testData {
		t.Run(td.name, func(t *testing.T) {
			tree := newTree(td.baseSeq)

			y := td.baseSeq
			a, b := y[0], y[len(y)-1]

			// Check edge value in forward iteration
			fwd := tree.NewIterator()
			if got, want := kvResultString(fwd.Next()), kvResultString(a, valType(a), true); got != want {
				t.Fatalf("fwd.Next() = %s; want %s", got, want)
			}

			// Check edge value in reverse iteration
			rev := tree.NewReverseIterator()
			if got, want := kvResultString(rev.Next()), kvResultString(b, valType(b), true); got != want {
				t.Fatalf("rev.Next() = %s; want %s", got, want)
			}

			// Apply the modification to the tree
			td.modifier(tree)

			// Check remaining iterator sequence against expected result
			fwdseq := getIterSeq(fwd)
			if !checkIterSeq(fwdseq, td.expectedFwdSeq) {
				t.Fatalf("unexpected forward iterator sequence %v; want %v", fwdseq, td.expectedFwdSeq)
			}
			revseq := getIterSeq(rev)
			if !checkIterSeq(revseq, td.expectedRevSeq) {
				t.Fatalf("unexpected reverse iterator sequence %v; want %v", revseq, td.expectedRevSeq)
			}
		})
	}

}

// Iterating over an empty tree should not return any associations.
func TestIterEmpty(t *testing.T) {
	tree := newTree(nil)
	iters := [2]*iterType{
		tree.NewIterator(),
		tree.NewReverseIterator(),
	}

	for i, iter := range iters {
		if got, want := kvResultString(iter.Next()), kvResultString(0, 0, false); got != want {
			t.Fatalf("iter%d.Get() = %v; want %v", i, got, want)
		}
	}
}

// Closed iterators should be invalidated.
func TestIterClose(t *testing.T) {
	tree := newTree([]keyType{1, 2, 3})
	iters := [2]*iterType{
		tree.NewIterator(),
		tree.NewReverseIterator(),
	}

	// Iterators should become invalidated when canceled
	for i, iter := range iters {
		iter.Close()
		if got, want := kvResultString(iter.Next()), kvResultString(0, 0, false); got != want {
			t.Fatalf("iter.Close: iter%d.Get() = %v; want %v", i, got, want)
		}
	}
}

// Removing from an empty tree should work.
func TestRemoveFromEmptyTree(t *testing.T) {
	tree := newTree(nil)
	const wantLength = 0
	if length := tree.Length(); length != wantLength {
		t.Fatalf("tree.Length() = %v; want %v", length, wantLength)
	}
	tree.Remove(1)
	if length := tree.Length(); length != wantLength {
		t.Fatalf("tree.Length() = %v; want %v", length, wantLength)
	}
}

// Removing a non-existing association should have no observable effects.
func TestRemoveNonExisting(t *testing.T) {
	want := []keyType{1, 2, 3, 5}
	tree := newTree(want)
	tree.Remove(4)
	got := getIterSeq(tree.NewIterator())
	if !checkIterSeq(got, want) {
		t.Fatalf("tree.Remove() -> got sequence %v; want %v", got, want)
	}
}

// Clearing a tree should remove all associations, calling the release function
// for each association when doing so and invalidate all iterators.
func TestClear(t *testing.T) {
	seq := []keyType{1, 2, 3, 4, 5, 6, 7, 8, 9}
	tree := newTree(seq)

	iters := [2]*iterType{
		tree.NewIterator(),
		tree.NewReverseIterator(),
	}

	var released []assoc
	tree.Clear(func(k keyType, v valType) {
		released = append(released, assoc{key: k, val: v})
	})

	if !checkIterSeq(released, seq) {
		t.Fatalf("unexpected release sequence %v; want %v", released, seq)
	}

	// Iterators should be invalidated since all data was removed.
	for i, iter := range iters {
		if got, want := kvResultString(iter.Next()), kvResultString(0, 0, false); got != want {
			t.Fatalf("tree.Clear: iter%d.Get() = %v; want %v", i, got, want)
		}
	}

	// The length should be zero.
	if got, want := tree.Length(), 0; got != want {
		t.Fatalf("tree.Clear: tree.Length() = %v; want %v", got, want)
	}
}

// Apply should visit all tree associations in the correct order.
func TestApply(t *testing.T) {
	seq := []keyType{1, 2, 3, 4, 5, 6, 7, 8, 9}
	tree := newTree(seq)

	var visited []assoc
	tree.Apply(func(k keyType, v valType) {
		visited = append(visited, assoc{key: k, val: v})
	})

	if !checkIterSeq(visited, seq) {
		t.Fatalf("unexpected visited sequence %v; want %v", visited, seq)
	}
}

// Length should reflect the number of associations in a tree.
func TestLength(t *testing.T) {
	seq := []keyType{1, 2, 3}
	tree := newTree(nil)
	count := 0

	// Testing that length is updated by adding associations.
	for _, k := range seq {
		if n := tree.Length(); n != count {
			t.Fatalf("tree.Len() = %d; want %d", n, count)
		}
		tree.Add(k, valType(k))
		count++
	}

	// Testing that length is updated by removing associations.
	for _, k := range seq {
		if n := tree.Length(); n != count {
			t.Fatalf("tree.Len() = %d; want %d", n, count)
		}
		tree.Remove(k)
		count--
	}

	// Testing the length after clearing a tree is done by TestClear.
}

// Find should return expected results.
func TestFind(t *testing.T) {
	tree := newTree([]keyType{2, 5, 6, 7, 10})

	testData := []struct {
		name string
		key  keyType
		want string
	}{
		{"Find(NonExisting)", 1, "0,false"},
		{"Find(NonExisting)", 4, "0,false"},
		{"Find(NonExisting)", 8, "0,false"},
		{"Find(NonExisting)", 11, "0,false"},
		{"Find(Existing)", 2, "2,true"},
		{"Find(Existing)", 6, "6,true"},
		{"Find(Existing)", 10, "10,true"},
	}

	for i, td := range testData {
		t.Run(fmt.Sprintf("%s/%d", td.name, i), func(t *testing.T) {
			if got, want := vResultString(tree.Find(td.key)), td.want; got != want {
				t.Fatalf("tree.FindOp(%d) = %s; want %s", td.key, got, want)
			}
		})
	}

	testData2 := []struct {
		name string
		find func(keyType) (keyType, valType, bool)
		key  keyType
		want string
	}{
		{"FindEqualOrLesser(NonExisting)", tree.FindEqualOrLesser, 11, "10,10,true"},
		{"FindEqualOrLesser(NonExisting)", tree.FindEqualOrLesser, 9, "7,7,true"},
		{"FindEqualOrLesser(NonExisting)", tree.FindEqualOrLesser, 4, "2,2,true"},
		{"FindEqualOrLesser(NonExisting)", tree.FindEqualOrLesser, 1, "0,0,false"},
		{"FindEqualOrLesser(Existing)", tree.FindEqualOrLesser, 2, "2,2,true"},
		{"FindEqualOrLesser(Existing)", tree.FindEqualOrLesser, 6, "6,6,true"},
		{"FindEqualOrLesser(Existing)", tree.FindEqualOrLesser, 10, "10,10,true"},
		{"FindEqualOrGreater(NonExisting)", tree.FindEqualOrGreater, 11, "0,0,false"},
		{"FindEqualOrGreater(NonExisting)", tree.FindEqualOrGreater, 8, "10,10,true"},
		{"FindEqualOrGreater(NonExisting)", tree.FindEqualOrGreater, 3, "5,5,true"},
		{"FindEqualOrGreater(NonExisting)", tree.FindEqualOrGreater, 1, "2,2,true"},
		{"FindEqualOrGreater(Existing)", tree.FindEqualOrGreater, 2, "2,2,true"},
		{"FindEqualOrGreater(Existing)", tree.FindEqualOrGreater, 6, "6,6,true"},
		{"FindEqualOrGreater(Existing)", tree.FindEqualOrGreater, 10, "10,10,true"},
	}

	for i, td := range testData2 {
		t.Run(fmt.Sprintf("%s/%d", td.name, i), func(t *testing.T) {
			if got, want := kvResultString(td.find(td.key)), td.want; got != want {
				t.Fatalf("tree.FindOp(%d) = %s; want %s", td.key, got, want)
			}
		})
	}
}

// FindLowest should return the association with the lowest key.
func TestFindLowest(t *testing.T) {
	tree := newTree(nil)
	if got, want := kvResultString(tree.FindLowest()), kvResultString(0, 0, false); got != want {
		t.Fatalf("tree.FindLowest() = %v; want %v", got, want)
	}

	tree = newTree([]keyType{1, 2, 3, 4, 5})
	if got, want := kvResultString(tree.FindLowest()), kvResultString(1, 1, true); got != want {
		t.Fatalf("tree.FindLowest() = %v; want %v", got, want)
	}
}

func TestFindHighest(t *testing.T) {
	tree := newTree(nil)
	if got, want := kvResultString(tree.FindHighest()), kvResultString(0, 0, false); got != want {
		t.Fatalf("tree.FindHighest() = %v; want %v", got, want)
	}

	tree = newTree([]keyType{1, 2, 3, 4, 5})
	if got, want := kvResultString(tree.FindHighest()), kvResultString(5, 5, true); got != want {
		t.Fatalf("tree.FindHighest() = %v; want %v", got, want)
	}
}

func newTree(keys []keyType, options ...treeOptionType) *treeType {
	return bulkInsert(avltree.New(math.CompareOrdered[keyType], options...), keys)
}

func bulkInsert(tree *treeType, keys []keyType) *treeType {
	for _, k := range keys {
		// Because keyType and valType are distinct types that are not
		// assignable to each other without casts we insert the key as
		// the value and rely on the type system to ensure that the two
		// are not mixed up. Tests can then assert that key = value when
		// testing APIs that return both.
		tree.Add(k, valType(k))
	}
	return tree
}

func bulkRemove(tree *treeType, keys []keyType) {
	for _, k := range keys {
		tree.Remove(k)
	}
}

func getIterSeq(iter *iterType) (seq []assoc) {
	for k, v, ok := iter.Next(); ok; k, v, ok = iter.Next() {
		seq = append(seq, assoc{k, v})
	}
	return
}

func checkIterSeq(assocs []assoc, expected []keyType) bool {
	if len(assocs) != len(expected) {
		return false
	}
	for i, assoc := range assocs {
		if e := expected[i]; assoc.key != e || assoc.val != valType(e) {
			return false
		}
	}
	return true
}

// Value result to string
func vResultString(v valType, ok bool) string {
	return fmt.Sprintf("%v,%v", v, ok)
}

// Key value result to string
func kvResultString(k keyType, v valType, ok bool) string {
	return fmt.Sprintf("%v,%v,%v", k, v, ok)
}

type someKeys [10]keyType

// Returns true on success, false on error (sequence finished).
func permute(dst, src *someKeys, seq int) bool {
	alen := len(src)

	// Factorial of alen
	fact := 1
	for i := 2; i < alen; i++ {
		fact *= i
	}

	// Out of range?
	if (seq / alen) >= fact {
		return false
	}

	*dst = *src

	for i := 0; i < (alen - 1); i++ {
		tmpi := (seq / fact) % (alen - i)
		tmp := dst[i+tmpi]

		for j := i + tmpi; j > i; j-- {
			dst[j] = dst[j-1]
		}

		dst[i] = tmp
		fact /= (alen - (i + 1))
	}

	return true
}
