# GoDS

Go Data Structures, version 2.

Compared to version 1 the APIs has been revised to take advantage of generic
data types introduced in Go 1.18.

## avltree

The AVL tree is converted from a C implementation originating from
https://web.archive.org/web/20070212102708/http://eternallyconfuzzled.com/tuts/datastructures/jsw_tut_avl.aspx
which is in the public domain.

I have used the C version in many projects and trust it but this Go conversion
is not yet battle proven. I believe the conversion is robust based on test
results.

### Design Choices

* Non thread safe. Use external mutexes to achieve concurrency safe operations.
* Iterators are internal and tree operations slow down when they are in use.
* Iterators are automatically advanced when the node they are parked on is removed.

## iter

An alternative way to use iterators that is ergonomic with Go's limited form of
while loop. It implements the scanner idiom popularized by bufio.Scanner.

## list (circular double linked list)

An intrusive circular double linked list. The most useful property is that a
list node can remove itself from any list without having a reference to one.
This is useful to implement certain algorithms such as timers or priority queues
where the list a list node is stored in may be unknown.

## math

Mostly simple utility functions such as abs, min and max.
