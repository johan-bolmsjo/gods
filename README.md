# GoDS

Go Data Structures.

## AVL Tree (avltree)

The AVL tree is converted from a C implementation originating from
<http://www.eternallyconfuzzled.com> which is in the public domain.

I have used the C version in many projects and trust it but this Go conversion
is not yet battle proven. I believe the conversion is robust based on test
results.

### Design Choices

* Non thread safe. Use external mutexes to achieve concurrency safe operations.
* Keys are stored in the data that is stored in the tree (not K+V).
* Iterators are internal and tree operations slow down when they are in use.
* Iterators are automatically advanced when the node they are parked on is removed.

### Gotchas

Never modify the key of data that is inserted in a tree as that invalidates the
ordering invariant of the tree.

Don't forget to call Cancel on iterators that don't iterate over the full range
of the tree or they will be linked to the tree forever. The iterator API is a
bit clunky and not a good fit for Go. It's a straight conversion from the C
version of the tree.

### Performance Comparison with C Version

Currently as of Go 1.11 it seems to be around 2.25 times slower than the C
version of the same tree when running the brute force invariant tests.

## Circular Double Linked List (list)

A very simple circular double linked list with possibilities to shoot oneself in
the foot. With this reduced safety comes some interesting algorithmic
possibilities.
