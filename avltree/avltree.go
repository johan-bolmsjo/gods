// Based on code originally written by:
// Julienne Walker, http://eternallyconfuzzled.com

package avltree

import (
	"github.com/johan-bolmsjo/gods/list"
	"sync"
)

// Data stored in the tree.
type Data interface{}

// Key of data stored in the tree.
type Key interface{}

// GetKey obtains the key from data stored in the tree.
type GetKey func(Data) Key

// CmpKey compares two keys and return a value less than, equal to, or greater
// than zero if lhs is found, respectively, to be less than, to match, or be
// greater than rhs.
type CmpKey func(lhs, rhs Key) int

const maxTreeHeight = 36

// Tree is an AVL tree.
type Tree struct {
	root   *node
	elems  int // Number of elements in tree
	getKey GetKey
	cmpKey CmpKey
	iters  list.Elem
}

// Iter is an iterator handle used to iterate over data stored in the tree.
type Iter struct {
	elem   list.Elem            // List element to make it linkable to tree iterator list
	tree   *Tree                // Tree iterator belongs to
	curr   *node                // Current node
	path   [maxTreeHeight]*node // Traversal path
	top    int                  // Top of stack
	dir    direction            // Direction of movement
	update bool                 // Update path before moving
}

// Scanner wraps an iterator and provides an API that is more convenient to use
// with Go's limited form of while loops.
type Scanner struct {
	iter *Iter
	data Data
}

type node struct {
	balance int      // Balance factor
	link    [2]*node //Left and right links.
	data    Data
}

// *** Tree ***

// New creates an AVL tree using the supplied getKey and cmpKey functions.
func New(getKey GetKey, cmpKey CmpKey) *Tree {
	tree := &Tree{
		getKey: getKey,
		cmpKey: cmpKey,
	}
	tree.iters.Init(nil)
	return tree
}

// Insert data into tree.
// Returns inserted data and true or existing data associated with getKey(data) and false.
func (tree *Tree) Insert(data Data) (Data, bool) {
	// Empty tree case
	if tree.root == nil {
		tree.root = newNode(data)
		tree.elems++
		return data, true
	}

	key := tree.getKey(data)

	// Set up false tree root to ease maintenance
	var head node
	t := &head
	t.link[directionRight] = tree.root

	var dir direction
	var s *node    // Place to rebalance and parent
	var p, q *node // Iterator and save pointer

	// Search down the tree, saving rebalance points
	for s, p = t.link[directionRight], t.link[directionRight]; ; p = q {
		cmp := tree.cmpKey(tree.getKey(p.data), key)
		if cmp == 0 {
			return p.data, false
		}

		dir = directionFromBool(cmp < 0)
		if q = p.link[dir]; q == nil {
			break
		}

		if q.balance != 0 {
			t = p
			s = q
		}
	}

	q = newNode(data)
	p.link[dir] = q

	// Update balance factors
	for p = s; p != q; p = p.link[dir] {
		dir = directionFromBool(tree.cmpKey(tree.getKey(p.data), key) < 0)
		p.balance += dir.balance()
	}

	q = s // Save rebalance point for parent fix

	// Rebalance if necessary
	if iabs(s.balance) > 1 {
		dir = directionFromBool(tree.cmpKey(tree.getKey(s.data), key) < 0)
		s = s.insertBalance(dir)
	}

	// Fix parent
	if q == head.link[directionRight] {
		tree.root = s
	} else {
		t.link[directionFromBool(q == t.link[directionRight])] = s
	}

	// Mark all iterators for path update
	for e := tree.iters.Next(); e != &tree.iters; e = e.Next() {
		iter := e.Value.(*Iter)
		iter.update = true
	}

	tree.elems++
	return data, true
}

// Remove data associated with key from tree.
// Returns the removed data or nil if none found.
func (tree *Tree) Remove(key Key) Data {
	if tree.root == nil {
		return nil
	}

	curr := tree.root
	var up [maxTreeHeight]*node
	var upd [maxTreeHeight]direction
	var top int

	// Search down tree and save path
	for {
		if curr == nil {
			return nil
		}

		cmp := tree.cmpKey(tree.getKey(curr.data), key)
		if cmp == 0 {
			break
		}

		// Push direction and node onto stack
		upd[top] = directionFromBool(cmp < 0)
		up[top] = curr
		top++

		curr = curr.link[upd[top-1]]
	}

	// Remove the node
	if curr.link[directionLeft] == nil || curr.link[directionRight] == nil {
		// Which child is non-nil?
		dir := directionFromBool(curr.link[directionLeft] == nil)

		// Fix parent
		if top != 0 {
			up[top-1].link[upd[top-1]] = curr.link[dir]
		} else {
			tree.root = curr.link[dir]
		}
	} else {
		// Find the inorder successor
		heir := curr.link[directionRight]

		// Save this path too
		upd[top] = directionRight
		up[top] = curr
		top++

		for heir.link[directionLeft] != nil {
			upd[top] = directionLeft
			up[top] = heir
			top++
			heir = heir.link[directionLeft]
		}

		// Swap data
		t := curr.data
		curr.data = heir.data
		heir.data = t

		// Unlink successor and fix parent
		up[top-1].link[directionFromBool(up[top-1] == curr)] = heir.link[directionRight]
		curr = heir
	}

	// Walk back up the search path
	var done bool

	for top--; top >= 0 && !done; top-- {
		// Update balance factors
		up[top].balance += upd[top].inverseBalance()

		// Terminate or rebalance as necessary
		if iabs(up[top].balance) == 1 {
			break
		} else if iabs(up[top].balance) > 1 {
			up[top], done = up[top].removeBalance(upd[top])

			// Fix parent
			if top != 0 {
				up[top-1].link[upd[top-1]] = up[top]
			} else {
				tree.root = up[0]
			}
		}
	}

	// Update iterators
	for e := tree.iters.Next(); e != &tree.iters; e = e.Next() {
		iter := e.Value.(*Iter)

		// All iterators need their path updated
		iter.update = true

		// Iterators positioned on the removed node need update performed now
		if iter.curr == curr {
			iter.update = false
			if iter.buildPathNext() == nil {
				// This one fell of the edge
				e = e.Prev()
				iter.Cancel()
			}
		}
	}

	data := curr.data
	curr.release(nil)
	tree.elems--
	return data
}

// Clear removes all data from the tree and invalidates all iterators.
// The releaseData function (if non nil) is called on removed data.
func (tree *Tree) Clear(releaseData func(Data)) {
	curr := tree.root

	// Destruction by rotation
	for curr != nil {
		var save *node

		if curr.link[directionLeft] == nil {
			// Remove node
			save = curr.link[directionRight]
			curr.release(releaseData)
		} else {
			// Rotate right
			save = curr.link[directionLeft]
			curr.link[directionLeft] = save.link[directionRight]
			save.link[directionRight] = curr
		}

		curr = save
	}

	tree.root = nil
	tree.elems = 0

	for tree.iters.IsLinked() {
		tree.iters.Next().Value.(*Iter).Cancel()
	}
}

// Len returns the number of elements in the tree.
func (tree *Tree) Len() int {
	return tree.elems
}

// Find data associated with key.
// Returns found data or nil.
func (tree *Tree) Find(key Key) Data {
	curr := tree.root

	for curr != nil {
		cmp := tree.cmpKey(tree.getKey(curr.data), key)
		if cmp == 0 {
			break
		}
		curr = curr.link[directionFromBool(cmp < 0)]
	}

	if curr != nil {
		return curr.data
	}
	return nil
}

// Find data associated with key or data whose key is immediately lesser.
// Returns found data or nil.
func (tree *Tree) FindLe(key Key) Data {
	curr := tree.root
	var lesser *node

	for curr != nil {
		cmp := tree.cmpKey(tree.getKey(curr.data), key)
		if cmp == 0 {
			break
		}
		if cmp < 0 {
			lesser = curr
		}
		curr = curr.link[directionFromBool(cmp < 0)]
	}

	if curr != nil {
		return curr.data
	} else if lesser != nil {
		return lesser.data
	}
	return nil
}

// Find data associated with key or data whose key is immediately greater.
// Returns found data or nil.
func (tree *Tree) FindGe(key Key) Data {
	curr := tree.root
	var greater *node

	for curr != nil {
		cmp := tree.cmpKey(tree.getKey(curr.data), key)
		if cmp == 0 {
			break
		}
		if cmp > 0 {
			greater = curr
		}
		curr = curr.link[directionFromBool(cmp < 0)]
	}

	if curr != nil {
		return curr.data
	} else if greater != nil {
		return greater.data
	}
	return nil
}

// Front returns the data of the leftmost element in the tree or nil if tree is empty.
func (tree *Tree) Front() Data {
	return tree.edgeNode(directionLeft)
}

// Back returns the data of the rightmost element in the tree or nil if tree is empty.
func (tree *Tree) Back() Data {
	return tree.edgeNode(directionRight)
}

// Apply calls the given function on all data stored in the tree.
func (tree *Tree) Apply(f func(data Data)) {
	s := NewScanner(tree.Iterate())
	for s.Scan() {
		f(s.Data())
	}
}

// Iterate creates an iterator positioned on the leftmost element in the tree.
// The iterator will move to the right when advanced.
func (tree *Tree) Iterate() *Iter {
	return tree.iterate(directionRight)
}

// Iterate creates an iterator positioned on the rightmost element in the tree.
// The iterator will move to the left when advanced.
func (tree *Tree) IterateReverse() *Iter {
	return tree.iterate(directionLeft)
}

func (tree *Tree) edgeNode(dir direction) Data {
	node := tree.root
	if node == nil {
		return nil
	}

	for node.link[dir] != nil {
		node = node.link[dir]
	}

	return node.data
}

func (tree *Tree) iterate(dir direction) *Iter {
	iter := &Iter{tree: tree, dir: dir}
	iter.elem.Init(iter)

	if iter.buildPathStart() != nil {
		tree.iters.LinkNext(&iter.elem)
	}
	return iter
}

// Validate tree invariants.
// A valid tree should always be balanced and sorted.
func (tree *Tree) Validate() (balanced, sorted bool) {
	balanced = true
	sorted = true

	if tree.root != nil {
		tree.validateNode(tree.root, &balanced, &sorted, 0)
	}
	return
}

func (tree *Tree) validateNode(node *node, rvBalanced, rvSorted *bool, depth int) int {
	depth++
	var depthLink [2]int

	for dir := directionLeft; dir <= directionRight; dir++ {
		depthLink[dir] = depth

		if node.link[dir] != nil {
			cmp := tree.cmpKey(tree.getKey(node.link[dir].data), tree.getKey(node.data))
			if dir == directionFromBool(cmp < 0) {
				*rvSorted = false
			}
			depthLink[dir] = tree.validateNode(node.link[dir], rvBalanced, rvSorted, depth)
		}
	}

	if iabs(depthLink[directionLeft]-depthLink[directionRight]) > 1 {
		*rvBalanced = false
	}

	return imax(depthLink[directionLeft], depthLink[directionRight])
}

// *** Iterators ***

// Data returns the data of the element the iterator is positioned on.
func (iter *Iter) Data() Data {
	if !iter.elem.IsLinked() {
		return nil
	}
	return iter.curr.data
}

// Next returns the data of the element the iterator is positioned on and advance it.
func (iter *Iter) Next() Data {
	if !iter.elem.IsLinked() {
		return nil
	}

	if iter.update {
		iter.buildPathCurr()
		iter.update = false
	}

	data := iter.curr.data
	if iter.advance() == nil {
		iter.Cancel()
	}
	return data
}

// Cancel invalidates the iterator and removes its reference from the tree it's associated with.
// Cancel is automatically called for iterators that iterate over the full range of a tree.
func (iter *Iter) Cancel() {
	iter.elem.Unlink()

	// Clear pointers to avoid GC memory leaks.
	iter.tree = nil
	iter.curr = nil

	for i := range iter.path {
		iter.path[i] = nil
	}
}

// Move iterator according to iterator direction.
func (iter *Iter) advance() Data {
	dir := iter.dir

	if iter.curr.link[dir] != nil {
		// Continue down this branch
		iter.path[iter.top] = iter.curr
		iter.curr = iter.curr.link[dir]
		iter.top++

		for iter.curr.link[dir.other()] != nil {
			iter.path[iter.top] = iter.curr
			iter.curr = iter.curr.link[dir.other()]
			iter.top++
		}
	} else {
		// Move to the next branch
		var last *node

		for {
			if iter.top == 0 {
				iter.curr = nil
				break
			}

			iter.top--
			last = iter.curr
			iter.curr = iter.path[iter.top]

			if last != iter.curr.link[dir] {
				break
			}
		}
	}

	if iter.curr != nil {
		return iter.curr.data
	}
	return nil
}

// Build path to first or last entry depending on iterator direction.
func (iter *Iter) buildPathStart() Data {
	dir := iter.dir.other()

	iter.curr = iter.tree.root
	iter.top = 0

	if iter.curr != nil {
		for iter.curr.link[dir] != nil {
			iter.path[iter.top] = iter.curr
			iter.curr = iter.curr.link[dir]
			iter.top++
		}
		return iter.curr.data
	}
	return nil
}

// Build path to current node (should always be in tree).
func (iter *Iter) buildPathCurr() {
	tree := iter.tree
	key := tree.getKey(iter.curr.data)

	iter.curr = tree.root
	iter.top = 0

	cmp := tree.cmpKey(tree.getKey(iter.curr.data), key)
	for cmp != 0 {
		iter.path[iter.top] = iter.curr
		iter.curr = iter.curr.link[directionFromBool(cmp < 0)]
		iter.top++

		cmp = tree.cmpKey(tree.getKey(iter.curr.data), key)
	}
}

// Build path to node next to current node.
func (iter *Iter) buildPathNext() Data {
	tree := iter.tree
	key := tree.getKey(iter.curr.data)

	var match *node

	iter.curr = tree.root
	iter.top = 0

	for iter.curr != nil {
		dir := directionFromBool(tree.cmpKey(tree.getKey(iter.curr.data), key) < 0)
		if dir != iter.dir {
			// This node matched the direction criteria.
			match = iter.curr
		}
		iter.path[iter.top] = iter.curr
		iter.curr = iter.curr.link[dir]
		iter.top++
	}

	if match != nil {
		// Wind back path to best match.
		for iter.curr != match {
			iter.top--
			iter.curr = iter.path[iter.top]
		}
		return match.data
	}
	return nil
}

// *** Scanner ***

// NewScanner wraps an iterator and provides an alternative iterator API.
func NewScanner(iter *Iter) *Scanner {
	return &Scanner{iter: iter}
}

// Scan fetch the next value from the wrapped iterator and saves it internally.
// The function reports whether a value was obtained from the iterator which can
// then be read out using the Data method.
func (s *Scanner) Scan() bool {
	s.data = s.iter.Next()
	return s.data != nil
}

// Data returns the saved value from the Scan method.
func (scanner *Scanner) Data() Data {
	return scanner.data
}

// *** node ***

func newNode(data Data) *node {
	node := nodePool.Get().(*node)
	node.data = data
	return node
}

var nodePool = sync.Pool{New: func() interface{} { return new(node) }}

func (node *node) release(releaseData func(Data)) {
	if releaseData != nil {
		releaseData(node.data)
	}

	// Clear pointers to avoid GC memory leaks.
	node.link[directionLeft] = nil
	node.link[directionRight] = nil
	node.data = nil

	// Clear balance before putting node in pool.
	node.balance = 0
	nodePool.Put(node)
}

// *** node: Rotations ***

// Two way single rotation
func (root *node) single(dir direction) *node {
	odir := dir.other()
	save := root.link[odir]
	root.link[odir] = save.link[dir]
	save.link[dir] = root
	return save
}

// Two way double rotation.
func (root *node) double(dir direction) *node {
	odir := dir.other()
	save := root.link[odir].link[dir]
	root.link[odir].link[dir] = save.link[odir]
	save.link[odir] = root.link[odir]
	root.link[odir] = save

	save = root.link[odir]
	root.link[odir] = save.link[dir]
	save.link[dir] = root
	return save
}

// Adjust balance before double rotation.
func (root *node) adjustBalance(dir direction, bal int) {
	n1 := root.link[dir]
	n2 := n1.link[dir.other()]

	if n2.balance == 0 {
		root.balance = 0
		n1.balance = 0
	} else if n2.balance == bal {
		root.balance = -bal
		n1.balance = 0
	} else {
		// n2.balance == -bal
		root.balance = 0
		n1.balance = bal
	}
	n2.balance = 0
}

// Rebalance after insertion.
func (root *node) insertBalance(dir direction) *node {
	n := root.link[dir]
	bal := dir.balance()

	if n.balance == bal {
		root.balance, n.balance = 0, 0
		root = root.single(dir.other())
	} else {
		// n.balance == -bal
		root.adjustBalance(dir, bal)
		root = root.double(dir.other())
	}

	return root
}

// Rebalance after deletion.
func (root *node) removeBalance(dir direction) (rnode *node, done bool) {
	n := root.link[dir.other()]
	bal := dir.balance()

	if n.balance == -bal {
		root.balance = 0
		n.balance = 0
		root = root.single(dir)
	} else if n.balance == bal {
		root.adjustBalance(dir.other(), -bal)
		root = root.double(dir)
	} else {
		// n.balance == 0
		root.balance = -bal
		n.balance = bal
		root = root.single(dir)
		done = true
	}

	return root, done
}

// *** Misc ***

// direction select left or right node links.
type direction int8

const (
	directionLeft  direction = 0
	directionRight direction = 1
)

func (dir direction) other() direction {
	return dir ^ 1 // invert direction
}

func (dir direction) balance() int {
	if dir == directionLeft {
		return -1
	}
	return +1
}

func (dir direction) inverseBalance() int {
	if dir != directionLeft {
		return -1
	}
	return +1
}

func directionFromBool(b bool) direction {
	if b {
		return directionRight
	}
	return directionLeft
}

func iabs(i int) int {
	if i < 0 {
		return -i
	}
	return i
}

func imax(i, j int) int {
	if i > j {
		return i
	}
	return j
}
