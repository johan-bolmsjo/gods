// Based on code originally written by Julienne Walker in the public domain,
// https://web.archive.org/web/20070212102708/http://eternallyconfuzzled.com/tuts/datastructures/jsw_tut_avl.aspx

package avltree

import (
	"sync"

	"github.com/johan-bolmsjo/gods/v2/list"
	"github.com/johan-bolmsjo/gods/v2/math"
)

// Maximum tree height supported by a tree.
// This is a *large* tree, larger than reasonable.
const maxTreeHeight = 48

/******************************************************************************
 * Tree
 *****************************************************************************/

// Tree is an AVL tree.
type Tree[K, V any] struct {
	root        *node[K, V]
	length      int
	nodePool    *nodePool[K, V]
	compareKeys math.Comparator[K]
	iters       list.Node[*Iterator[K, V]]
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
	tree.iters.InitLinks()
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

	// Mark all iterators for path update
	for e := tree.iters.Next(); e != &tree.iters; e = e.Next() {
		iter := e.Value
		iter.update = true
	}

	tree.length++
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

	// Update iterators
	for e := tree.iters.Next(); e != &tree.iters; e = e.Next() {
		iter := e.Value

		// All iterators need their path updated
		iter.update = true

		// Iterators positioned on the removed node need update performed now
		if iter.curr == curr {
			iter.update = false
			if !iter.buildPathNext() {
				// This one fell of the edge
				e = e.Prev()
				iter.Close()
			}
		}
	}

	tree.nodePool.put(curr, nil)
	tree.length--
}

// Clear removes all associations from the tree and invalidates all iterators. A
// non-nil release function is called on each association in the tree. The
// release function must not fail. Remove each association by itself (for
// example by using an iterator) if it can fail and handle errors properly.
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

	for tree.iters.IsLinked() {
		tree.iters.Next().Value.Close()
	}
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

// Apply calls the supplied function for each association in the tree.
func (tree *Tree[K, V]) Apply(f func(K, V)) {
	iter := tree.NewIterator()
	for k, v, ok := iter.Next(); ok; k, v, ok = iter.Next() {
		f(k, v)
	}
}

// NewIterator creates an iterator that advances from low to high key values.
// Make sure to close the iterator by calling its Close method when done.
func (tree *Tree[K, V]) NewIterator() *Iterator[K, V] {
	return tree.iterator(directionRight)
}

// NewReverseIterator creates an iterator that advances from high to low key
// values. Make sure to close the iterator by calling its Close method when
// done.
func (tree *Tree[K, V]) NewReverseIterator() *Iterator[K, V] {
	return tree.iterator(directionLeft)
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

func (tree *Tree[K, V]) iterator(dir direction) *Iterator[K, V] {
	iter := &Iterator[K, V]{tree: tree, dir: dir}
	iter.listNode.InitLinks().Value = iter

	if iter.buildPathStart() {
		tree.iters.LinkNext(&iter.listNode)
	}
	return iter
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

	return math.MaxInteger(depthLink[directionLeft], depthLink[directionRight])
}

/******************************************************************************
 * Iterator
 *****************************************************************************/

// Iterator that is used to iterate over associations in a tree.
type Iterator[K, V any] struct {
	listNode list.Node[*Iterator[K, V]] // List node to make it linkable to tree iterator list
	tree     *Tree[K, V]                // Tree iterator belongs to
	curr     *node[K, V]                // Current node
	path     [maxTreeHeight]*node[K, V] // Traversal path
	top      int                        // Top of stack
	dir      direction                  // Direction of movement
	update   bool                       // Update path before moving
}

// Next returns the next association from the iterator. The zero values of K and
// V and false is returned if the iterator is not positioned on any association
// (such as when all associations has been visited). Close has been called when
// false is returned.
func (iter *Iterator[K, V]) Next() (K, V, bool) {
	if !iter.listNode.IsLinked() {
		return zeroAssoc[K, V]()
	}

	if iter.update {
		iter.buildPathCurr()
		iter.update = false
	}

	key, value := iter.curr.key, iter.curr.value
	if !iter.advance() {
		iter.Close()
	}
	return key, value, true
}

// Close invalidates the iterator and removes its reference from the tree it's
// associated with. It's safe to call the Next method on closed iterators.
func (iter *Iterator[K, V]) Close() {
	iter.listNode.Unlink()

	// Clear pointers to avoid GC memory leaks.
	iter.tree = nil
	iter.curr = nil
	for i := range iter.path {
		iter.path[i] = nil
	}
}

// Move iterator according to its recorded direction and report whether it fell
// over the edge.
func (iter *Iterator[K, V]) advance() bool {
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
		var last *node[K, V]

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

	return iter.curr != nil
}

// Build path to first or last association depending on iterator direction and
// report if it was successful.
func (iter *Iterator[K, V]) buildPathStart() bool {
	dir := iter.dir.other()

	iter.curr = iter.tree.root
	iter.top = 0

	if iter.curr != nil {
		for iter.curr.link[dir] != nil {
			iter.path[iter.top] = iter.curr
			iter.curr = iter.curr.link[dir]
			iter.top++
		}
		return true
	}
	return false
}

// Build path to current node (should always be in tree).
func (iter *Iterator[K, V]) buildPathCurr() {
	tree := iter.tree
	key := iter.curr.key

	iter.curr = tree.root
	iter.top = 0

	for cmp := tree.compareKeys(iter.curr.key, key); cmp != 0; cmp = tree.compareKeys(iter.curr.key, key) {
		iter.path[iter.top] = iter.curr
		iter.curr = iter.curr.link[directionOfBool(cmp < 0)]
		iter.top++
	}
}

// Build path to node next to current node and report whether it fell over the
// edge.
func (iter *Iterator[K, V]) buildPathNext() bool {
	tree := iter.tree
	key := iter.curr.key

	var match *node[K, V]

	iter.curr = tree.root
	iter.top = 0

	for iter.curr != nil {
		dir := directionOfBool(tree.compareKeys(iter.curr.key, key) < 0)
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
		return true
	}
	return false
}

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
