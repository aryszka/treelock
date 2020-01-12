# Treelock

Locking mechanism for tree structures.

## Features

- usable with any tree structure whose nodes can be addressed by their path
- RWMutex style read and write support
- locking for individual nodes or for complete subtrees
- fairness in the order of allowing operations to proceed that depend on the same nodes

## Documentation

See the package and method descriptions at: https://godoc.org/github.com/aryszka/treelock