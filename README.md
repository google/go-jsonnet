# go-jsonnet

[![GoDoc Widget]][GoDoc] [![Travis Widget]][Travis] [![Coverage Status Widget]][Coverage Status]

[GoDoc]: https://godoc.org/github.com/google/go-jsonnet
[GoDoc Widget]: https://godoc.org/github.com/google/go-jsonnet?status.png
[Travis]: https://travis-ci.org/google/go-jsonnet
[Travis Widget]: https://travis-ci.org/google/go-jsonnet.svg?branch=master
[Coverage Status Widget]: https://coveralls.io/repos/github/google/go-jsonnet/badge.svg?branch=master
[Coverage Status]: https://coveralls.io/github/google/go-jsonnet?branch=master

This is a port of [jsonnet](http://jsonnet.org/) to go.  It is very much a work in progress.

This implementation is largely based on the the [jsonnet C++ implementation](https://github.com/google/jsonnet).
The precise revision is
https://github.com/google/jsonnet/tree/27ddf2c2f7041c09316cf7c9ef13af9588fdd671 but when we reach
feature parity with that revision, we will chase up all the recent changes on the C++ side.

## Implementation Notes

We are generating some helper classes on types by using http://clipperhouse.github.io/gen/.  Do the following to regenerate these if necessary:

```
go get github.com/clipperhouse/gen
go get github.com/clipperhouse/set
go generate
```
