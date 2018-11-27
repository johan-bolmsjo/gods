package list

// Elem is a list element carrying some data.
// To form a practical list a sentinel element is used to represent the list head.
// The zero value is not a valid element.
type Elem struct {
	next, prev *Elem
	Value      interface{}
}

// New returns an initialized element on which list operations may be performed.
func New(value interface{}) *Elem {
	return new(Elem).Init(value)
}

// Init initializes an element so that list operations may be performed on it.
func (e *Elem) Init(value interface{}) *Elem {
	e.next, e.prev = e, e
	e.Value = value
	return e
}

// LinkNext links another element next to e.
func (e *Elem) LinkNext(other *Elem) {
	t := other.prev
	e.next.prev = t
	t.next = e.next
	other.prev = e
	e.next = other
}

// LinkPrev links another element previous to e.
func (e *Elem) LinkPrev(other *Elem) {
	t := other.prev
	e.prev.next = other
	t.next = e
	other.prev = e.prev
	e.prev = t
}

// Unlink removes e from list.
// It's safe to unlink an already unlinked element.
func (e *Elem) Unlink() {
	e.next.prev = e.prev
	e.prev.next = e.next
	e.next, e.prev = e, e
}

// Next returns the element next to e (which may be e itself).
func (e *Elem) Next() *Elem {
	return e.next
}

// Prev returns the element previous to e (which may be e itself).
func (e *Elem) Prev() *Elem {
	return e.prev
}

// IsLinked reports whether the element is linked to another element or not.
// If applied to a sentinel list head element it can be used to check if the list is empty.
func (e *Elem) IsLinked() bool {
	return e.next != e
}
