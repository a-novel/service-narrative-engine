// Package configtest holds the configuration fixtures the service's tests share. Only _test.go
// files import it.
//
// Go links every package a production file imports into the binary, so these fixtures stay out of
// it only while no production import path reaches this package.
package configtest
