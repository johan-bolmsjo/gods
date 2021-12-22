/*
Package list provides an intrusive circular double linked list. The most useful
property is that a list node can remove itself from any list without having a
reference to one. This is useful to implement certain algorithms such as timers
or priority queues where the list a list node is stored in may be unknown.

Illustration of element nodes forming a circular list. One of the elements is
dedicated to the role of list head and does not carry any valid data.

    +------------------------------------+
    |                                    |
    +--> +----+ --> +----+ --> +----+ -->+
         |Elem|     |Elem|     |Elem|
    +<-- +----+ <-- +----+ <-- +----+ <--+
    |                                    |
    +------------------------------------+
*/
package list
