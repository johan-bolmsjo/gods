package list_test

import (
	"github.com/johan-bolmsjo/gods/list"
	"testing"
)

type link struct {
	next, prev *list.Elem
}

func value(e *list.Elem) interface{} {
	if e == nil {
		return "nil"
	}
	return e.Value
}

func checkLinks(t *testing.T, e *list.Elem, l []link) {
	for i, v := range l {
		if e.Next() != v.next {
			t.Fatalf("expected next node of %v (index %d) to be %v; got %v",
				value(e), i, value(v.next), value(e.Next()))
		}
		if e.Prev() != v.prev {
			t.Fatalf("expected previous node of %v (index %d) to be %v; got %v",
				value(e), i, value(v.prev), value(e.Prev()))
		}
		e = e.Next()
	}
}

func TestLinkNext(t *testing.T) {
	var e [5]*list.Elem
	for i, _ := range e {
		e[i] = list.New(i)
	}

	// Link elements form a list
	h1 := e[0]
	h1.LinkNext(e[1])
	h1.LinkNext(e[2])

	// Link two multi element lists together
	h2 := e[3]
	h2.LinkNext(e[4])
	h1.LinkNext(h2)

	// Expected element order [0, 3, 4, 2, 1]
	checkLinks(t, h1, []link{
		link{e[3], e[1]},
		link{e[4], e[0]},
		link{e[2], e[3]},
		link{e[1], e[4]},
		link{e[0], e[2]},
	})
}

func TestLinkPrev(t *testing.T) {
	var e [5]*list.Elem
	for i, _ := range e {
		e[i] = list.New(i)
	}

	// Link elements form a list
	h1 := e[0]
	h1.LinkPrev(e[1])
	h1.LinkPrev(e[2])

	// Link two multi element lists together
	h2 := e[3]
	h2.LinkPrev(e[4])
	h1.LinkPrev(h2)

	// Expected element order [0, 1, 2, 3, 4]
	checkLinks(t, h1, []link{
		link{e[1], e[4]},
		link{e[2], e[0]},
		link{e[3], e[1]},
		link{e[4], e[2]},
		link{e[0], e[3]},
	})
}

func TestUnlink(t *testing.T) {
	var e [3]*list.Elem
	for i, _ := range e {
		e[i] = list.New(i)
	}

	h1 := e[0]

	t.Run("1", func(t *testing.T) {
		h1.LinkPrev(e[1])
		h1.LinkPrev(e[2])

		// Expected element order [0, 2]
		e[1].Unlink()
		checkLinks(t, h1, []link{
			link{e[2], e[2]},
			link{e[0], e[0]},
		})
	})

	t.Run("2", func(t *testing.T) {
		// Test that the unlinked element point to itself
		checkLinks(t, e[1], []link{
			link{e[1], e[1]},
		})
	})

	// Remove last element
	// Expected element order [0]
	//
	// Do it twice to make sure that unlinking an unlinked element has no effect.
	for i := 0; i < 2; i++ {
		t.Run("3", func(t *testing.T) {
			e[2].Unlink()
			checkLinks(t, h1, []link{
				link{e[0], e[0]},
			})
		})
	}
}

func TestIsLinked(t *testing.T) {
	e0, e1 := list.New(0), list.New(1)

	if e0.IsLinked() {
		t.Fatalf("e0.IsLinked() = true; want false")
	}

	e0.LinkPrev(e1)
	if !e0.IsLinked() {
		t.Fatalf("e0.IsLinked() = false; want true")
	}
	if !e1.IsLinked() {
		t.Fatalf("e1.IsLinked() = false; want true")
	}
}
