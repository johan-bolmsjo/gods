# GoDS

Go Data Structures, version 4.

Compared to version 3, the AVL tree API has minor changes. The Clear
method no longer accepts a release function as it had limitations in
that it was not allowed to fail. The FindLowest and FindHighest methods
are renamed to First and Last respectively.

Compared to version 2, the APIs has been revised to use the Go 1.23
iterators. The math MinInteger and MaxInteger functions has been
removed. Use the built-in min and max functions instead.

Compared to version 1, the APIs has been revised to take advantage of
generic data types introduced in Go 1.18.

## avltree

The AVL tree is converted from a C implementation originating from
https://web.archive.org/web/20070212102708/http://eternallyconfuzzled.com/tuts/datastructures/jsw_tut_avl.aspx
which is in the public domain.

I have used the C version in many projects and trust it but this Go
conversion is not yet battle proven. I believe the conversion is robust
based on test results.

### Design Choices

The AVL tree is not thread safe. Use external mutexes to achieve
concurrency safe operations.

## list (circular double linked list)

An intrusive circular double linked list. The most useful property is
that a list node can remove itself from any list without having a
reference to one. This is useful to implement certain algorithms such as
timers or priority queues where the list a list node is stored in may be
unknown.

## math

Mostly simple utility functions.
