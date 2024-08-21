# GoDS

Go Data Structures, version 3.

Compared to version 1, the APIs has been revised to take advantage of
generic data types introduced in Go 1.18.

Compared to version 2, the APIs has been revised to use the iterator API
introduced in Go 1.23. The math MinInteger and MaxInteger functions has
been removed. Use the built-in min and max functions instead.

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
