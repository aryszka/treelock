/*
Package treelock implements a read/write locking mechanism for arbitrary tree structures.

Node path

The package can be used with tree structures whose nodes can be referenced by their path. E.g. in case of a file
system, a file with the path /a/b/c can be locked with the treelock path of "a", "b", "c", while an empty
treelock path would mean locking the root: /.

Read and write

The package assumes that the nodes of the protected tree structure allow multiple concurrent read operations,
but only a single write operation at a given time. Therefore, locks acquired by ReadNode and ReadTree will allow
to proceed operations concurrently acquiring the same kind of locks over the affected nodes, but will block
operations that try to acquire locks with WriteNode or WriteTree, until each read lock has been released.
Conversely, locks acquired by WriteNode and WriteTree will block any subsequent operations on the affected nodes
until the lock is released.

Nodes and subtrees

It is possible to acquire locks only for individual nodes, when an operation needs to read or write data only of
one node. In these cases, only those subsequent operations are blocked that involve the locked node. When some
operations need to read a snapshot of a subtree, or carry out structural changes to existing nodes in the tree,
ReadTree or WriteTree need to be used. E.g. in case of a file system, when an operation needs to copy a
directory structure to under another path, it needs to acquire a ReadTree lock on the source directory, and a
WriteTree lock on the destination.

Fairness

Operations affecting the same nodes will be allowed to proceed in the same order as they requested the lock,
regardless of the type of the lock. Operations affecting independent nodes will be allowed to proceed as soon as
the affected node becomes available.
*/
package treelock
