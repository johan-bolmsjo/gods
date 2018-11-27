package avltree_test

import (
	"github.com/johan-bolmsjo/gods/avltree"
	"strconv"
	"testing"
)

// Brute force test of tree rotations triggered by inserting elements.
// Tree invariants are validated after each operation.
func TestInvariantsPermuteInsert(t *testing.T) {
	tree := newTree(nil)
	src := someInts{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	var dst someInts
	alen := len(src)

	seq := 0
	for permute(&dst, &src, seq) {
		for j := 0; j < alen; j++ {
			rdata, ok := tree.Insert(&dst[j])
			if !ok {
				t.Fatalf("Failed to insert data=%v, index=%d, insertSequence=%v, returnedData=%v",
					dst[j], j, dst, *rdata.(*int))
			}

			balanced, sorted := tree.Validate()
			if !balanced || !sorted {
				t.Fatalf("Invalid tree invariant: balanced=%v, sorted=%v, insertSequence=%v",
					balanced, sorted, dst)
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
	tree := newTree(nil)
	src := someInts{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	var dst someInts
	alen := len(src)

	seq := 0
	for permute(&dst, &src, seq) {
		for j := 0; j < alen; j++ {
			rdata, ok := tree.Insert(&src[j])
			if !ok {
				t.Fatalf("Failed to insert data=%v, index=%d, insertSequence=%v, returnedData=%v",
					src[j], j, src, *rdata.(*int))
			}
		}

		for j := 0; j < alen; j++ {
			rdata := tree.Remove(&dst[j])
			if *rdata.(*int) != dst[j] {
				t.Fatalf("Failed to remove data=%v, index=%d, removeSequence=%v, returnedData=%v",
					dst[j], j, dst, *rdata.(*int))
			}

			balanced, sorted := tree.Validate()
			if !balanced || !sorted {
				t.Fatalf("Invalid tree invariant: balanced=%v, sorted=%v, removeSequence=%v",
					balanced, sorted, dst)
			}
		}
		tree.Clear(nil)
		seq++
	}
	t.Logf("%d remove sequences tested", seq)
}

// Inserting data whose key already exist in the tree should return the old data.
func TestInsertExisting(t *testing.T) {
	tree := newTree(nil)

	a := 1
	ra, ok := tree.Insert(&a)
	if dtop(ra) != &a || !ok {
		t.Fatalf("tree.Insert() = (%p, %t); want (%p, %t)", dtop(ra), ok, &a, true)
	}

	b := 1
	rb, ok := tree.Insert(&b)

	if dtop(rb) != &a || ok {
		t.Fatalf("tree.Insert() = (%p, %t); want (%p, %t)", dtop(rb), ok, &a, false)
	}
}

// Test that iterators are updated by insert and remove operations.
func TestIteratorUpdate(t *testing.T) {
	testData := []struct {
		name           string
		modifier       func(t *avltree.Tree)
		baseData       []int
		expectedFwdSeq []int
		expectedRevSeq []int
	}{
		{
			"Insert",
			func(t *avltree.Tree) { bulkInsert(t, []int{2, 10}) },
			[]int{1, 3, 5, 7, 9, 11},
			[]int{3, 5, 7, 9, 10, 11},
			[]int{9, 7, 5, 3, 2, 1},
		},
		{
			"Remove",
			func(t *avltree.Tree) { bulkRemove(t, []int{1, 11}) },
			[]int{1, 3, 5, 7, 9, 11},
			[]int{3, 5, 7, 9},
			[]int{9, 7, 5, 3},
		},
		{
			"RemoveAndMove", // Iterator should be moved by remove operation
			func(t *avltree.Tree) { bulkRemove(t, []int{3, 9}) },
			[]int{1, 3, 5, 7, 9, 11},
			[]int{5, 7, 11},
			[]int{7, 5, 1},
		},
		{
			"RemoveAndInvalidate", // Iterator should fall off the edge and be invalidated
			func(t *avltree.Tree) { bulkRemove(t, []int{1, 3, 5}) },
			[]int{1, 3, 5},
			[]int{},
			[]int{},
		},
	}

	for _, td := range testData {
		t.Run(td.name, func(t *testing.T) {
			tree := newTree(td.baseData)

			y := td.baseData
			a, b, c, d := y[0], y[1], y[len(y)-2], y[len(y)-1]

			fwd := tree.Iterate()
			if x, y := dtos(fwd.Next()), itos(a); x != y {
				t.Fatalf("fwd.Next() = %s; want %s", x, y)
			}
			if x, y := dtos(fwd.Data()), itos(b); x != y {
				t.Fatalf("fwd.Data() = %s; want %s", x, y)
			}

			rev := tree.IterateReverse()
			if x, y := dtos(rev.Next()), itos(d); x != y {
				t.Fatalf("rev.Next() = %s; want %s", x, y)
			}
			if x, y := dtos(rev.Data()), itos(c); x != y {
				t.Fatalf("rev.Data() = %s; want %s", x, y)
			}

			td.modifier(tree)

			fwdseq := getIterSeq(fwd)
			revseq := getIterSeq(rev)

			if !intSlicesEqual(fwdseq, td.expectedFwdSeq) {
				t.Fatalf("unexpected forward iterator sequence %v; want %v", fwdseq, td.expectedFwdSeq)
			}
			if !intSlicesEqual(revseq, td.expectedRevSeq) {
				t.Fatalf("unexpected reverse iterator sequence %v; want %v", revseq, td.expectedRevSeq)
			}
		})
	}

}

func TestIterEmpty(t *testing.T) {
	tree := newTree(nil)
	iters := [2]*avltree.Iter{
		tree.Iterate(),
		tree.IterateReverse(),
	}

	for i, v := range iters {
		if d := v.Data(); d != nil {
			t.Fatalf("iter%d.Data() = %v; want %v", i, d, nil)
		}
	}
}

func TestIterCancel(t *testing.T) {
	tree := newTree([]int{1, 2, 3})
	iters := [2]*avltree.Iter{
		tree.Iterate(),
		tree.IterateReverse(),
	}

	// Iterators should become invalidated when canceled
	for i, v := range iters {
		v.Cancel()
		if d := v.Data(); d != nil {
			t.Fatalf("iter.Cancel: iter%d.Data() = %v; want %v", i, d, nil)
		}
	}
}

func TestRemoveFromEmptyTree(t *testing.T) {
	tree := newTree(nil)
	a := int(1)
	if r := tree.Remove(&a); r != nil {
		t.Fatalf("tree.Remove() = %v; want %v", r, nil)
	}
}

func TestRemoveNonExisting(t *testing.T) {
	tree := newTree([]int{1, 2, 3, 5})
	a := int(4)
	if r := tree.Remove(&a); r != nil {
		t.Fatalf("tree.Remove() = %v; want %v", r, nil)
	}
}

func TestClear(t *testing.T) {
	seq := []int{1, 2, 3, 4, 5, 6, 7, 8, 9}
	tree := newTree(seq)

	iters := [2]*avltree.Iter{
		tree.Iterate(),
		tree.IterateReverse(),
	}

	var released []int
	tree.Clear(func(d avltree.Data) {
		released = append(released, *d.(*int))
	})

	if !intSlicesEqual(seq, released) {
		t.Fatalf("unexpected release sequence %v; want %v", released, seq)
	}

	// Iterators should have become invalidated since all data was removed.
	for i, v := range iters {
		if d := v.Data(); d != nil {
			t.Fatalf("tree.Clear: iter%d.Data() = %v; want %v", i, d, nil)
		}
	}
}

func TestApply(t *testing.T) {
	seq := []int{1, 2, 3, 4, 5, 6, 7, 8, 9}
	tree := newTree(seq)

	var visited []int
	tree.Apply(func(d avltree.Data) {
		visited = append(visited, *d.(*int))
	})

	if !intSlicesEqual(seq, visited) {
		t.Fatalf("unexpected release sequence %v; want %v", visited, seq)
	}
}

func TestLen(t *testing.T) {
	tree := newTree(nil)
	count := 0
	for _, v := range []int{1, 2, 3} {
		if n := tree.Len(); n != count {
			t.Fatalf("tree.Len() = %d; want %d", n, count)
		}
		bulkInsert(tree, []int{v})
		count++
	}
}

func TestFind(t *testing.T) {
	tree := newTree([]int{2, 5, 6, 7, 10})
	find := func(k int) avltree.Data { return tree.Find(&k) }
	findLe := func(k int) avltree.Data { return tree.FindLe(&k) }
	findGe := func(k int) avltree.Data { return tree.FindGe(&k) }

	testData := []struct {
		name string
		find func(k int) avltree.Data
		in   int
		want string
	}{
		{"Find(NonExisting)", find, 1, noData},
		{"Find(NonExisting)", find, 4, noData},
		{"Find(NonExisting)", find, 8, noData},
		{"Find(NonExisting)", find, 11, noData},
		{"Find(Existing)", find, 2, "2"},
		{"Find(Existing)", find, 6, "6"},
		{"Find(Existing)", find, 10, "10"},
		{"FindLe(NonExisting)", findLe, 11, "10"},
		{"FindLe(NonExisting)", findLe, 9, "7"},
		{"FindLe(NonExisting)", findLe, 4, "2"},
		{"FindLe(NonExisting)", findLe, 1, noData},
		{"FindLe(Existing)", findLe, 2, "2"},
		{"FindLe(Existing)", findLe, 6, "6"},
		{"FindLe(Existing)", findLe, 10, "10"},
		{"FindGe(NonExisting)", findGe, 11, noData},
		{"FindGe(NonExisting)", findGe, 8, "10"},
		{"FindGe(NonExisting)", findGe, 3, "5"},
		{"FindGe(NonExisting)", findGe, 1, "2"},
		{"FindGe(Existing)", findGe, 2, "2"},
		{"FindGe(Existing)", findGe, 6, "6"},
		{"FindGe(Existing)", findGe, 10, "10"},
	}

	for _, td := range testData {
		t.Run(td.name, func(t *testing.T) {
			if out := dtos(td.find(td.in)); out != td.want {
				t.Fatalf("tree.FindOp(%d) = %s; want %s", td.in, out, td.want)
			}
		})
	}
}

func TestFront(t *testing.T) {
	tree := newTree(nil)
	if r := tree.Front(); r != nil {
		t.Fatalf("tree.Front() = %v; want %v", r, nil)
	}

	tree = newTree([]int{1, 2, 3, 4, 5})
	if r := dtos(tree.Front()); r != "1" {
		t.Fatalf("tree.Front() = %s; want %s", r, "1")
	}
}

func TestBack(t *testing.T) {
	tree := newTree(nil)
	if r := tree.Back(); r != nil {
		t.Fatalf("tree.Back() = %v; want %v", r, nil)
	}

	tree = newTree([]int{1, 2, 3, 4, 5})
	if r := dtos(tree.Back()); r != "5" {
		t.Fatalf("tree.Back() = %s; want %s", r, "5")
	}
}

func newTree(data []int) *avltree.Tree {
	return bulkInsert(avltree.New(
		func(data avltree.Data) avltree.Key { return data }, // Just storing ints in the tree
		func(a, b avltree.Key) int { return *a.(*int) - *b.(*int) },
	), data)
}

func bulkInsert(tree *avltree.Tree, data []int) *avltree.Tree {
	for i := range data {
		_, ok := tree.Insert(&data[i])
		if !ok {
			die()
		}
	}
	return tree
}

func bulkRemove(tree *avltree.Tree, data []int) {
	for i := range data {
		r := tree.Remove(&data[i])
		if *r.(*int) != data[i] {
			die()
		}
	}
}

func getIterSeq(iter *avltree.Iter) (seq []int) {
	s := avltree.NewScanner(iter)
	for s.Scan() {
		seq = append(seq, *s.Data().(*int))
	}
	return
}

func intSlicesEqual(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

const noData = "<nil>"

// Data to string
// Solves problem of dereferencing nil interface in tests.
func dtos(d avltree.Data) string {
	if d == nil {
		return noData
	}
	return itos(*d.(*int))
}

// Data to pointer
// Solves problem of dereferencing nil interface in tests.
func dtop(d avltree.Data) *int {
	if d == nil {
		return nil
	}
	return d.(*int)
}

func itos(i int) string {
	return strconv.Itoa(i)
}

type someInts [10]int

// Returns true on success, false on error (sequence finished).
func permute(dst, src *someInts, seq int) bool {
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

// Used in helpers to prep tree for other tests.
// Should not fail but I guess it could if something is really broken.
func die() {
	panic("basic assumption violated")
}
