// Based on code originally written by Julienne Walker in the public domain,
// https://web.archive.org/web/20070212102708/http://eternallyconfuzzled.com/tuts/datastructures/jsw_tut_avl.aspx

package avltree

import (
	"iter"
	"sync"

	"github.com/johan-bolmsjo/gods/v2/math"
)

// Maximum tree height supported by a tree.
// This is a *large* tree, larger than reasonable.
const maxTreeHeight = 48

/******************************************************************************
 * Tree Options
 *****************************************************************************/

type TreeOption[K, V any] func(*Tree[K, V])

// WithSyncPool creates a tree option to use a sync.Pool to reuse nodes to
// reduce pressure on the garbage collector. It may improve performance for
// trees with lots of updates. The option holds an instance of a sync.Pool that
// may be used by multiple trees in multiple go routines in a safe manner.
func WithSyncPool[K, V any]() TreeOption[K, V] {
	nodePool := newNodePool[K, V]()
	return func(tree *Tree[K, V]) {
		tree.nodePool = nodePool
	}
}

/******************************************************************************
 * Tree
 *****************************************************************************/

// Tree is an AVL tree.
type Tree[K, V any] struct {
	root        *node[K, V]
	length      int
	nodePool    *nodePool[K, V]
	compareKeys math.Comparator[K]
	generation  uint64 // Generation of tree mutation (used for iterator invalidation)
}

// New creates an AVL tree using the supplied compare function and tree options.
// Avoid using pointer keys that are dereferenced by the compare function as
// modifying such keys outside of the tree invalidates the ordering invariant of
// the tree.
func New[K, V any](compareKeys math.Comparator[K], options ...TreeOption[K, V]) *Tree[K, V] {
	tree := &Tree[K, V]{
		compareKeys: compareKeys,
	}
	for _, option := range options {
		option(tree)
	}
	return tree
}

// Add association between key and value to the tree. Any existing association
// for key is overwritten with key and value.
func (tree *Tree[K, V]) Add(key K, value V) {
	// Empty tree case
	if tree.root == nil {
		tree.root = tree.nodePool.get()
		tree.root.key = key
		tree.root.value = value
		tree.length++
		tree.generation++
		return
	}

	// Set up false tree root to ease maintenance
	var head node[K, V]
	t := &head
	t.link[directionRight] = tree.root

	var dir direction
	var s *node[K, V]    // Place to rebalance and parent
	var p, q *node[K, V] // Iterator and save pointer

	// Search down the tree, saving rebalance points
	for s, p = t.link[directionRight], t.link[directionRight]; ; p = q {
		cmp := tree.compareKeys(p.key, key)
		if cmp == 0 {
			// Update association
			p.key, p.value = key, value
			return
		}

		dir = directionOfBool(cmp < 0)
		if q = p.link[dir]; q == nil {
			break
		}

		if q.balance != 0 {
			t = p
			s = q
		}
	}

	q = tree.nodePool.get()
	q.key, q.value = key, value
	p.link[dir] = q

	// Update balance factors
	for p = s; p != q; p = p.link[dir] {
		dir = directionOfBool(tree.compareKeys(p.key, key) < 0)
		p.balance += dir.balance()
	}

	q = s // Save rebalance point for parent fix

	// Rebalance if necessary
	if math.AbsSigned(s.balance) > 1 {
		dir = directionOfBool(tree.compareKeys(s.key, key) < 0)
		s = s.insertBalance(dir)
	}

	// Fix parent
	if q == head.link[directionRight] {
		tree.root = s
	} else {
		t.link[directionOfBool(q == t.link[directionRight])] = s
	}

	tree.length++
	tree.generation++
}

// Remove any association with key from tree.
func (tree *Tree[K, V]) Remove(key K) {
	if tree.root == nil {
		return
	}

	curr := tree.root
	var up [maxTreeHeight]*node[K, V]
	var upd [maxTreeHeight]direction
	var top int

	// Search down tree and save path
	for {
		if curr == nil {
			return
		}

		cmp := tree.compareKeys(curr.key, key)
		if cmp == 0 {
			break
		}

		// Push direction and node onto stack
		upd[top] = directionOfBool(cmp < 0)
		up[top] = curr
		top++

		curr = curr.link[upd[top-1]]
	}

	// Remove the node
	if curr.link[directionLeft] == nil || curr.link[directionRight] == nil {
		// Which child is non-nil?
		dir := directionOfBool(curr.link[directionLeft] == nil)

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

		// Swap associations
		tmpKey, tmpValue := curr.key, curr.value
		curr.key, curr.value = heir.key, heir.value
		heir.key, heir.value = tmpKey, tmpValue

		// Unlink successor and fix parent
		up[top-1].link[directionOfBool(up[top-1] == curr)] = heir.link[directionRight]
		curr = heir
	}

	// Walk back up the search path
	var done bool

	for top--; top >= 0 && !done; top-- {
		// Update balance factors
		up[top].balance += upd[top].inverseBalance()

		// Terminate or rebalance as necessary
		if math.AbsSigned(up[top].balance) == 1 {
			break
		} else if math.AbsSigned(up[top].balance) > 1 {
			up[top], done = up[top].removeBalance(upd[top])

			// Fix parent
			if top != 0 {
				up[top-1].link[upd[top-1]] = up[top]
			} else {
				tree.root = up[0]
			}
		}
	}

	tree.nodePool.put(curr, nil)
	tree.length--
	tree.generation++
}

// Clear removes all associations from the tree. A non-nil release function is called on
// each association in the tree. The release function must not fail. Remove each
// association by itself if the release operation can fail and handle errors properly.
func (tree *Tree[K, V]) Clear(release func(K, V)) {
	curr := tree.root

	// Destruction by rotation
	for curr != nil {
		var save *node[K, V]

		if curr.link[directionLeft] == nil {
			// Remove node
			save = curr.link[directionRight]
			tree.nodePool.put(curr, release)
		} else {
			// Rotate right
			save = curr.link[directionLeft]
			curr.link[directionLeft] = save.link[directionRight]
			save.link[directionRight] = curr
		}
		curr = save
	}

	tree.root = nil
	tree.length = 0
	tree.generation++
}

// Length returns the number of associations in the tree.
func (tree *Tree[K, V]) Length() int {
	return tree.length
}

// Find value associated with key. Returns the found value and true or the zero
// value of V and false if no assocation was found.
func (tree *Tree[K, V]) Find(key K) (V, bool) {
	curr := tree.root
	for curr != nil {
		cmp := tree.compareKeys(curr.key, key)
		if cmp == 0 {
			break
		}
		curr = curr.link[directionOfBool(cmp < 0)]
	}
	if curr != nil {
		return curr.value, true
	}
	return zeroValue[V]()
}

// FindEqualOrLesser returns the association that match key or the association
// with the immediately lesser key and true. The zero values of K and V and
// false is returned if no assocation was found.
func (tree *Tree[K, V]) FindEqualOrLesser(key K) (K, V, bool) {
	var lesser *node[K, V]

	curr := tree.root
	for curr != nil {
		cmp := tree.compareKeys(curr.key, key)
		if cmp == 0 {
			break
		}
		if cmp < 0 {
			lesser = curr
		}
		curr = curr.link[directionOfBool(cmp < 0)]
	}
	if curr != nil {
		return curr.key, curr.value, true
	} else if lesser != nil {
		return lesser.key, lesser.value, true
	}
	return zeroAssoc[K, V]()
}

// FindEqualOrGreater returns the association that match key or the immediately
// greater association and true. The zero values of K and V and false is
// returned if no assocation was found.
func (tree *Tree[K, V]) FindEqualOrGreater(key K) (K, V, bool) {
	var greater *node[K, V]

	curr := tree.root
	for curr != nil {
		cmp := tree.compareKeys(curr.key, key)
		if cmp == 0 {
			break
		}
		if cmp > 0 {
			greater = curr
		}
		curr = curr.link[directionOfBool(cmp < 0)]
	}
	if curr != nil {
		return curr.key, curr.value, true
	} else if greater != nil {
		return greater.key, greater.value, true
	}
	return zeroAssoc[K, V]()
}

// FindLowest returns the association with the lowest key and true. The zero value
// of K and V and false is returned if the tree is empty.
func (tree *Tree[K, V]) FindLowest() (K, V, bool) {
	return tree.edgeNode(directionLeft)
}

// FindHighest returns the association with the highest key and true. The zero value
// of K and V and false is returned if the tree is empty.
func (tree *Tree[K, V]) FindHighest() (K, V, bool) {
	return tree.edgeNode(directionRight)
}

func (tree *Tree[K, V]) edgeNode(dir direction) (K, V, bool) {
	node := tree.root
	if node == nil {
		return zeroAssoc[K, V]()
	}
	for node.link[dir] != nil {
		node = node.link[dir]
	}
	return node.key, node.value, true
}

// Validate tree invariants. A valid tree should always be balanced and sorted.
func (tree *Tree[K, V]) Validate() (balanced, sorted bool) {
	balanced = true
	sorted = true

	if tree.root != nil {
		tree.validateNode(tree.root, &balanced, &sorted, 0)
	}
	return
}

func (tree *Tree[K, V]) validateNode(node *node[K, V], rvBalanced, rvSorted *bool, depth int) int {
	depth++
	var depthLink [2]int

	for dir := directionLeft; dir <= directionRight; dir++ {
		depthLink[dir] = depth

		if node.link[dir] != nil {
			cmp := tree.compareKeys(node.link[dir].key, node.key)
			if dir == directionOfBool(cmp < 0) {
				*rvSorted = false
			}
			depthLink[dir] = tree.validateNode(node.link[dir], rvBalanced, rvSorted, depth)
		}
	}

	if math.AbsSigned(depthLink[directionLeft]-depthLink[directionRight]) > 1 {
		*rvBalanced = false
	}

	return max(depthLink[directionLeft], depthLink[directionRight])
}

/******************************************************************************
 * Iterator
 *****************************************************************************/

// All returns a "left to right" iterator over the tree. Any tree mutation while
// iterating, except for updating the value of an existing association, invalidates the
// iterator and iteration terminates prematurely.
func (tree *Tree[K, V]) All() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		if tree.root != nil {
			tree.allRecurse(yield, tree.root, tree.generation)
		}
	}
}

func (tree *Tree[K, V]) allRecurse(yield func(K, V) bool, node *node[K, V], startGeneration uint64) bool {
	if child := node.link[directionLeft]; child != nil && !tree.allRecurse(yield, child, startGeneration) {
		return false
	}
	if tree.generation != startGeneration || !yield(node.key, node.value) {
		return false
	}
	if child := node.link[directionRight]; child != nil && !tree.allRecurse(yield, child, startGeneration) {
		return false
	}
	return true
}

// Backward returns a "right to left" iterator over the tree. Any tree mutation while
// iterating, except for updating the value of an existing association, invalidates the
// iterator and iteration terminates prematurely.
func (tree *Tree[K, V]) Backward() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		if tree.root != nil {
			tree.backwardRecurse(yield, tree.root, tree.generation)
		}
	}
}

func (tree *Tree[K, V]) backwardRecurse(yield func(K, V) bool, node *node[K, V], startGeneration uint64) bool {
	if child := node.link[directionRight]; child != nil && !tree.backwardRecurse(yield, child, startGeneration) {
		return false
	}
	if tree.generation != startGeneration || !yield(node.key, node.value) {
		return false
	}
	if child := node.link[directionLeft]; child != nil && !tree.backwardRecurse(yield, child, startGeneration) {
		return false
	}
	return true
}

/******************************************************************************
 * Node
 *****************************************************************************/

type node[K, V any] struct {
	link    [2]*node[K, V] //Left and right links.
	balance int            // Balance factor
	key     K
	value   V
}

// Two way single rotation
func (root *node[K, V]) singleRotation(dir direction) *node[K, V] {
	odir := dir.other()
	save := root.link[odir]
	root.link[odir] = save.link[dir]
	save.link[dir] = root
	return save
}

// Two way double rotation.
func (root *node[K, V]) doubleRotation(dir direction) *node[K, V] {
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
func (root *node[K, V]) adjustBalance(dir direction, bal int) {
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
func (root *node[K, V]) insertBalance(dir direction) *node[K, V] {
	n := root.link[dir]
	bal := dir.balance()

	if n.balance == bal {
		root.balance, n.balance = 0, 0
		root = root.singleRotation(dir.other())
	} else {
		// n.balance == -bal
		root.adjustBalance(dir, bal)
		root = root.doubleRotation(dir.other())
	}

	return root
}

// Rebalance after deletion.
func (root *node[K, V]) removeBalance(dir direction) (rnode *node[K, V], done bool) {
	n := root.link[dir.other()]
	bal := dir.balance()

	if n.balance == -bal {
		root.balance = 0
		n.balance = 0
		root = root.singleRotation(dir)
	} else if n.balance == bal {
		root.adjustBalance(dir.other(), -bal)
		root = root.doubleRotation(dir)
	} else {
		// n.balance == 0
		root.balance = -bal
		n.balance = bal
		root = root.singleRotation(dir)
		done = true
	}

	return root, done
}

/******************************************************************************
 * Node pool
 *****************************************************************************/

// A type safe wrapper around sync.Pool.
type nodePool[K, V any] struct {
	pool sync.Pool
}

// newNodePool allocates a new node pool holding nodes with keys of type K and
// values of type V.
func newNodePool[K, V any]() *nodePool[K, V] {
	return &nodePool[K, V]{pool: sync.Pool{New: func() any { return new(node[K, V]) }}}
}

// Get node from pool. The pool may be nil in which case a normal allocation is
// performed.
func (pool *nodePool[K, V]) get() *node[K, V] {
	if pool != nil {
		return pool.pool.Get().(*node[K, V])
	}
	return &node[K, V]{}
}

// Return node to pool. The pool may be nil in which case the release function
// is called but no other action is performed.
func (pool *nodePool[K, V]) put(node *node[K, V], release func(K, V)) {
	if release != nil {
		release(node.key, node.value)
	}

	if pool != nil {
		// Clear pointers to avoid GC memory leaks as the node will be put in a
		// pool for reuse. Unless this is done this reachable object may keep
		// other objects alive which could otherwise be garbage collected.
		node.link[directionLeft] = nil
		node.link[directionRight] = nil

		// Keys and values can also be or contain pointers.
		node.key, _ = zeroValue[K]()
		node.value, _ = zeroValue[V]()

		// Clear balance before putting node in pool.
		node.balance = 0

		pool.pool.Put(node)
	}
}

/******************************************************************************
 * Miscellaneous
 *****************************************************************************/

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

func directionOfBool(b bool) direction {
	if b {
		return directionRight
	}
	return directionLeft
}

func zeroValue[V any]() (v V, ok bool) {
	return
}

func zeroAssoc[K, V any]() (k K, v V, ok bool) {
	return
}
