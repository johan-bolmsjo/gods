package list

// Node is a list node carrying a value of type T. A sentinel node is used to
// represent the list head. The zero value is not a valid node as its prev and
// next pointers must be initialized.
type Node[T any] struct {
	next, prev *Node[T]
	Value      T // Value carried by the node that may be accessed directly.
}

// New returns a node on which list operations may be performed.
func New[T any]() *Node[T] {
	return new(Node[T]).InitLinks()
}

// InitLinks initializes a node so that list operations may be performed on it.
// This method should be used to initialize list nodes for which the user
// manages memory. It should not be used when New was used to allocate a node
func (node *Node[T]) InitLinks() *Node[T] {
	node.next, node.prev = node, node
	return node
}

// LinkNext links other next to node.
func (node *Node[T]) LinkNext(other *Node[T]) {
	t := other.prev
	node.next.prev = t
	t.next = node.next
	other.prev = node
	node.next = other
}

// LinkPrev links other previous to node.
func (node *Node[T]) LinkPrev(other *Node[T]) {
	t := other.prev
	node.prev.next = other
	t.next = node
	other.prev = node.prev
	node.prev = t
}

// Unlink removes node from its list. It's safe to unlink unlinked nodes.
func (node *Node[T]) Unlink() {
	node.next.prev = node.prev
	node.prev.next = node.next
	node.next, node.prev = node, node
}

// Next returns the node next to node. This may be node itself if unlinked and pointing to itself.
func (node *Node[T]) Next() *Node[T] {
	return node.next
}

// Prev returns the node previous to node. This may be node itself if unlinked and pointing to itself.
func (node *Node[T]) Prev() *Node[T] {
	return node.prev
}

// IsLinked reports whether node is linked to another node. The method reports
// if a list is non-empty when applied to a list head node.
func (node *Node[T]) IsLinked() bool {
	return node.next != node
}
