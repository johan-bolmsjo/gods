package avltree_test

import (
	"fmt"
	"slices"
	"testing"

	"github.com/johan-bolmsjo/gods/v2/avltree"
	"github.com/johan-bolmsjo/gods/v2/math"
)

type keyType int
type valType int
type treeType = avltree.Tree[keyType, valType]
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

// Adding a key that already exist shall overwrite the existing association.
func TestAddExisting(t *testing.T) {
	for _, newTreeF := range newTreeFuncs {
		tree := newTreeF(nil)

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
}

// Removing from an empty tree shall have no observable effect.
func TestRemoveFromEmptyTree(t *testing.T) {
	for _, newTreeF := range newTreeFuncs {
		tree := newTreeF(nil)
		const wantLength = 0
		if length := tree.Length(); length != wantLength {
			t.Fatalf("tree.Length() = %v; want %v", length, wantLength)
		}
		tree.Remove(1)
		if length := tree.Length(); length != wantLength {
			t.Fatalf("tree.Length() = %v; want %v", length, wantLength)
		}
	}
}

// Removing a non-existing association shall have no observable effects.
func TestRemoveNonExisting(t *testing.T) {
	for _, newTreeF := range newTreeFuncs {
		want := []keyType{1, 2, 3, 5}
		tree := newTreeF(want)
		tree.Remove(4)
		got := collectAll(tree)
		if !checkSequence(got, want) {
			t.Fatalf("post: tree.Remove() -> %v; want %v", got, want)
		}
	}
}

// Clearing a tree shall remove all associations and calling the release function
// for each association when doing so.
func TestClear(t *testing.T) {
	for _, newTreeF := range newTreeFuncs {
		keys := []keyType{1, 2, 3, 4, 5, 6, 7, 8, 9}
		tree := newTreeF(keys)

		var released []assoc
		tree.Clear(func(k keyType, v valType) {
			released = append(released, assoc{k, v})
		})

		if !checkSequence(released, keys) {
			t.Fatalf("unexpected release sequence %v; want %v", released, keys)
		}

		// The length should be zero.
		if got, want := tree.Length(), 0; got != want {
			t.Fatalf("tree.Clear: tree.Length() = %v; want %v", got, want)
		}
	}
}

// Length shall reflect the number of associations in a tree.
func TestLength(t *testing.T) {
	for _, newTreeF := range newTreeFuncs {
		keys := []keyType{1, 2, 3}
		tree := newTreeF(nil)
		count := 0

		// Testing that length is updated by adding associations.
		for _, k := range keys {
			if n := tree.Length(); n != count {
				t.Fatalf("tree.Len() = %d; want %d", n, count)
			}
			tree.Add(k, valType(k))
			count++
		}

		// Testing that length is updated by removing associations.
		for _, k := range keys {
			if n := tree.Length(); n != count {
				t.Fatalf("tree.Len() = %d; want %d", n, count)
			}
			tree.Remove(k)
			count--
		}

		// Testing the length after clearing a tree is done by TestClear.
	}
}

// Find shall return expected results.
func TestFind(t *testing.T) {
	for _, newTreeF := range newTreeFuncs {
		tree := newTreeF([]keyType{2, 5, 6, 7, 10})

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
}

// FindLowest shall return the association with the lowest key.
func TestFindLowest(t *testing.T) {
	for _, newTreeF := range newTreeFuncs {
		tree := newTreeF(nil)
		if got, want := kvResultString(tree.FindLowest()), kvResultString(0, 0, false); got != want {
			t.Fatalf("tree.FindLowest() = %v; want %v", got, want)
		}

		tree = newTreeF([]keyType{1, 2, 3, 4, 5})
		if got, want := kvResultString(tree.FindLowest()), kvResultString(1, 1, true); got != want {
			t.Fatalf("tree.FindLowest() = %v; want %v", got, want)
		}
	}
}

// FindHighest shall return the association with the highest key.
func TestFindHighest(t *testing.T) {
	for _, newTreeF := range newTreeFuncs {
		tree := newTreeF(nil)
		if got, want := kvResultString(tree.FindHighest()), kvResultString(0, 0, false); got != want {
			t.Fatalf("tree.FindHighest() = %v; want %v", got, want)
		}

		tree = newTreeF([]keyType{1, 2, 3, 4, 5})
		if got, want := kvResultString(tree.FindHighest()), kvResultString(5, 5, true); got != want {
			t.Fatalf("tree.FindHighest() = %v; want %v", got, want)
		}
	}
}

// All shall return an iterator returning elements from lowest to highest key stored in the tree.
func TestIterateAll(t *testing.T) {
	for _, newTreeF := range newTreeFuncs {
		keys := []keyType{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
		tree := newTreeF(keys)

		var assocs []assoc
		for k, v := range tree.All() {
			assocs = append(assocs, assoc{k, v})
		}
		if !checkSequence(assocs, keys) {
			t.Fatalf("range tree.All() = %v; want %v", assocs, keys)
		}
	}
}

// Backward shall return an iterator returning elements from highest to lowest key stored in the tree.
func TestIterateBackward(t *testing.T) {
	for _, newTreeF := range newTreeFuncs {
		keys := []keyType{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
		tree := newTreeF(keys)

		var assocs []assoc
		for k, v := range tree.Backward() {
			assocs = append(assocs, assoc{k, v})
		}
		slices.Reverse(keys)
		if !checkSequence(assocs, keys) {
			t.Fatalf("range tree.Backward() = %v; want %v", assocs, keys)
		}
	}
}

// Breaking out of a range loop shall terminate iteration.
func TestIterateBreak(t *testing.T) {
	for _, newTreeF := range newTreeFuncs {
		keys := []keyType{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
		want := []keyType{0, 1, 2, 3, 4, 5}
		tree := newTreeF(keys)

		var assocs []assoc
		for k, v := range tree.All() {
			assocs = append(assocs, assoc{k, v})
			if k == 5 {
				break
			}
		}
		if !checkSequence(assocs, want) {
			t.Fatalf("range tree.All() = %v; want %v", assocs, want)
		}
	}
}

// Clear shall terminate any ongoing tree iteration.
func TestIterateInvalidateClear(t *testing.T) {
	for _, newTreeF := range newTreeFuncs {
		keys := []keyType{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
		want := []keyType{0, 1, 2, 3, 4, 5}
		tree := newTreeF(keys)

		var assocs []assoc
		var clearAssocs []assoc
		for k, v := range tree.All() {
			assocs = append(assocs, assoc{k, v})
			if k == 5 {
				// Iteration is terminated by modification
				tree.Clear(func(k keyType, v valType) {
					clearAssocs = append(clearAssocs, assoc{k, v})
				})
			}
		}
		if la, lw := tree.Length(), 0; la != lw {
			t.Fatalf("tree.Length() = %v; want %v", la, lw)
		}
		if !checkSequence(assocs, want) {
			t.Fatalf("range tree.All() = %v; want %v", assocs, want)
		}
		if !checkSequence(clearAssocs, keys) {
			t.Fatalf("range tree.Clear() = %v; want %v", clearAssocs, keys)
		}
	}
}

// Add new elements shall terminate any ongoing tree iteration.
func TestIterateInvalidateAddNew(t *testing.T) {
	for _, newTreeF := range newTreeFuncs {
		// Test adding elements to the left and right of current tree edges.
		for _, addKey := range []keyType{-1, 10} {
			keys := []keyType{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
			want := []keyType{0, 1, 2, 3, 4, 5}
			tree := newTreeF(keys)

			var assocs []assoc
			for k, v := range tree.All() {
				assocs = append(assocs, assoc{k, v})
				if k == 5 {
					tree.Add(addKey, valType(addKey))
				}
			}
			if la, lw := tree.Length(), 11; la != lw {
				t.Fatalf("tree.Length() = %v; want %v", la, lw)
			}
			if !checkSequence(assocs, want) {
				t.Fatalf("range tree.All() = %v; want %v", assocs, want)
			}
		}
	}
}

// Add in update case shall not terminate any ongoing tree iteration.
func TestIterateInvalidateAddUpdate(t *testing.T) {
	for _, newTreeF := range newTreeFuncs {
		keys := []keyType{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
		tree := newTreeF(keys)

		var assocs []assoc
		for k, v := range tree.All() {
			assocs = append(assocs, assoc{k, v})
			tree.Add(k, v+1)
		}
		if la, lw := tree.Length(), 10; la != lw {
			t.Fatalf("tree.Length() = %v; want %v", la, lw)
		}
		if !checkSequence(assocs, keys) {
			t.Fatalf("range tree.All() = %v; want %v", assocs, keys)
		}
	}
}

// Remove existing element shall terminate any ongoing tree iteration.
func TestIterateInvalidateRemoveExisting(t *testing.T) {
	for _, newTreeF := range newTreeFuncs {
		// Test removing left, current and right elements compared to iterator position.
		for _, removeKey := range []keyType{2, 5, 7} {
			keys := []keyType{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
			want := []keyType{0, 1, 2, 3, 4, 5}
			tree := newTreeF(keys)

			var assocs []assoc
			for k, v := range tree.All() {
				assocs = append(assocs, assoc{k, v})
				if k == 5 {
					tree.Remove(removeKey)
				}
			}
			if la, lw := tree.Length(), 9; la != lw {
				t.Fatalf("tree.Length() = %v; want %v", la, lw)
			}
			if !checkSequence(assocs, want) {
				t.Fatalf("range tree.All() = %v; want %v", assocs, want)
			}
		}
	}
}

// Remove non-existing element shall not terminate any ongoing tree iteration.
func TestIterateInvalidateRemoveNonExisting(t *testing.T) {
	for _, newTreeF := range newTreeFuncs {
		keys := []keyType{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
		tree := newTreeF(keys)

		var assocs []assoc
		for k, v := range tree.All() {
			assocs = append(assocs, assoc{k, v})
			if k == 5 {
				tree.Remove(-1)
				tree.Remove(10)
			}
		}
		if la, lw := tree.Length(), 10; la != lw {
			t.Fatalf("tree.Length() = %v; want %v", la, lw)
		}
		if !checkSequence(assocs, keys) {
			t.Fatalf("range tree.All() = %v; want %v", assocs, keys)
		}

	}
}

// Test that iteration is terminated as expected when mutating tree in nested iteration.
func TestIterateInvalidateNestedMutate(t *testing.T) {
	for _, newTreeF := range newTreeFuncs {
		keys := []keyType{0, 1, 2}
		want := []keyType{0, 0, 1, 2, 1, 0, 1}
		tree := newTreeF(keys)

		var assocs []assoc
		for ko, vo := range tree.All() {
			assocs = append(assocs, assoc{ko, vo})
			for ki, vi := range tree.All() {
				assocs = append(assocs, assoc{ki, vi})
				if ko == 1 && ki == 1 {
					tree.Remove(2)
				}
			}
		}
		if la, lw := tree.Length(), 2; la != lw {
			t.Fatalf("tree.Length() = %v; want %v", la, lw)
		}
		if !checkSequence(assocs, want) {
			t.Fatalf("range tree.All() = %v; want %v", assocs, want)
		}
	}
}

type newTreeFunc func([]keyType) *treeType

var newTreeFuncs = []newTreeFunc{
	func(keys []keyType) *treeType { return newTree(keys) },
	func(keys []keyType) *treeType { return newTree(keys, avltree.WithSyncPool[keyType, valType]()) },
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

func collectAll(tree *treeType) (assocs []assoc) {
	for k, v := range tree.All() {
		assocs = append(assocs, assoc{k, v})
	}
	return
}

func checkSequence(assocs []assoc, expected []keyType) bool {
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
