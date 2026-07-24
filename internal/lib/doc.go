// Package lib holds the primitives that depend on nothing else inside this service: the one
// outbound HTTP client every provider call shares, and the ticker background loops run on. Every
// layer above may import it, and it imports none of them.
//
// It stays deliberately small. Before adding to it, check whether the standard library or a
// dependency already covers the need, and whether the code belongs beside the single package that
// uses it instead.
package lib
