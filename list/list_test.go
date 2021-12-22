package list_test

import (
	"fmt"
	"testing"

	"github.com/johan-bolmsjo/gods/v2/list"
)

type link[T any] struct {
	next, prev *list.Node[T]
}

func valueOfNode[T any](e *list.Node[T]) any {
	if e == nil {
		return "nil"
	}
	return e.Value
}

func checkLinks[T any](t *testing.T, head *list.Node[T], order []link[T]) {
	node := head
	for i, v := range order {
		if node.Next() != v.next {
			t.Fatalf("expected next node of %v (index %d) to be %v; got %v",
				valueOfNode(node), i, valueOfNode(v.next), valueOfNode(node.Next()))
		}
		if node.Prev() != v.prev {
			t.Fatalf("expected previous node of %v (index %d) to be %v; got %v",
				valueOfNode(node), i, valueOfNode(v.prev), valueOfNode(node.Prev()))
		}
		node = node.Next()
	}
}

func TestLinkNext(t *testing.T) {
	var nodes [5]*list.Node[int]
	for i := range nodes {
		nodes[i] = list.New[int]()
		nodes[i].Value = i
	}

	// Link nodes to form a list
	head1 := nodes[0]
	head1.LinkNext(nodes[1])
	head1.LinkNext(nodes[2])

	// Link two multi node lists together
	head2 := nodes[3]
	head2.LinkNext(nodes[4])
	head1.LinkNext(head2)

	// Expected list node order [0, 3, 4, 2, 1]
	checkLinks(t, head1, []link[int]{
		{nodes[3], nodes[1]},
		{nodes[4], nodes[0]},
		{nodes[2], nodes[3]},
		{nodes[1], nodes[4]},
		{nodes[0], nodes[2]},
	})
}

func TestLinkPrev(t *testing.T) {
	var nodes [5]*list.Node[int]
	for i := range nodes {
		nodes[i] = list.New[int]()
		nodes[i].Value = i
	}

	// Link node to form a list
	head1 := nodes[0]
	head1.LinkPrev(nodes[1])
	head1.LinkPrev(nodes[2])

	// Link two multi node lists together
	head2 := nodes[3]
	head2.LinkPrev(nodes[4])
	head1.LinkPrev(head2)

	// Expected list node order [0, 1, 2, 3, 4]
	checkLinks(t, head1, []link[int]{
		{nodes[1], nodes[4]},
		{nodes[2], nodes[0]},
		{nodes[3], nodes[1]},
		{nodes[4], nodes[2]},
		{nodes[0], nodes[3]},
	})
}

func TestUnlink(t *testing.T) {
	var nodes [3]*list.Node[int]
	for i := range nodes {
		nodes[i] = list.New[int]()
		nodes[i].Value = i
	}

	head := nodes[0]

	t.Run("1", func(t *testing.T) {
		head.LinkPrev(nodes[1])
		head.LinkPrev(nodes[2])

		// Expected list node order [0, 2]
		nodes[1].Unlink()
		checkLinks(t, head, []link[int]{
			{nodes[2], nodes[2]},
			{nodes[0], nodes[0]},
		})
	})

	t.Run("2", func(t *testing.T) {
		// Test that the unlinked node point to itself
		checkLinks(t, nodes[1], []link[int]{
			{nodes[1], nodes[1]},
		})
	})

	// Remove last node
	// Expected list node order [0]
	//
	// Do it twice to make sure that unlinking an unlinked node has no effect.
	for i := 0; i < 2; i++ {
		t.Run(fmt.Sprintf("3/%d", i), func(t *testing.T) {
			nodes[2].Unlink()
			checkLinks(t, head, []link[int]{
				{nodes[0], nodes[0]},
			})
		})
	}
}

func TestIsLinked(t *testing.T) {
	var node0, node1 list.Node[int]
	node0.InitLinks().Value = 0
	node1.InitLinks().Value = 1

	if node0.IsLinked() {
		t.Fatalf("e0.IsLinked() = true; want false")
	}

	node0.LinkPrev(&node1)
	if !node0.IsLinked() {
		t.Fatalf("e0.IsLinked() = false; want true")
	}
	if !node1.IsLinked() {
		t.Fatalf("e1.IsLinked() = false; want true")
	}
}
