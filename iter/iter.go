package iter

// Iterator produce values of type T.
type Iterator[T any] interface {
	// Next returns the next value from the iterator and true if valid
	// output was produced.
	Next() (T, bool)
}

// Scanner provides an API that is ergonomic with Go's limited form of while
// loop. Use the Scan method as the termination clause and the Result method in
// the loop body.
type Scanner[T any] struct {
	t T
	g Iterator[T]
}

// NewScanner creates a scanner that fetch values from the given iterator.
func NewScanner[T any](g Iterator[T]) *Scanner[T] {
	return &Scanner[T]{g: g}
}

// Scan gets the next item from its iterator and stores it for later retrieval.
// Reports weather the iterator produced output or not.
func (s *Scanner[T]) Scan() (ok bool) {
	s.t, ok = s.g.Next()
	return
}

// Result of the last successful Scan operation.
func (s *Scanner[T]) Result() T {
	return s.t
}

// PairIterator produce pairs of values of type T and U.
type PairIterator[T, U any] interface {
	Next() (T, U, bool)
}

// PairScanner provides an API that is ergonomic with Go's limited form of while
// loop. Use the Scan method as the termination clause and the Result method in
// the loop body.
type PairScanner[T, U any] struct {
	t T
	u U
	g PairIterator[T, U]
}

// NewPairScanner creates a scanner that fetch values from the given iterator.
func NewPairScanner[T, U any](g PairIterator[T, U]) *PairScanner[T, U] {
	return &PairScanner[T, U]{g: g}
}

// Scan gets the next item from its iterator and stores it for later retrieval.
// Reports weather the iterator produced output or not.
func (s *PairScanner[T, U]) Scan() (ok bool) {
	s.t, s.u, ok = s.g.Next()
	return
}

// Result of the last successful Scan operation.
func (s *PairScanner[T, U]) Result() (T, U) {
	return s.t, s.u
}
