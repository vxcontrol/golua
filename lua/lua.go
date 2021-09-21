// This package provides access to the excellent lua language interpreter from go code.
//
// Access to most of the functions in lua.h and lauxlib.h is provided as well as additional convenience functions to publish Go objects and functions to lua code.
//
// The documentation of this package is no substitute for the official lua documentation and in many instances methods are described only with the name of their C equivalent
package lua

/*
#cgo CFLAGS: -I ${SRCDIR} -I ${SRCDIR}/lua
#cgo linux,386 LDFLAGS: -L ${SRCDIR}/lib/linux32 -lbundle -lluajit -lm -ldl
#cgo linux,amd64 LDFLAGS: -L ${SRCDIR}/lib/linux64 -lbundle -lluajit -lm -ldl
#cgo darwin,386 LDFLAGS: -L ${SRCDIR}/lib/osx32 -lbundle -lluajit
#cgo darwin,amd64 LDFLAGS: -L ${SRCDIR}/lib/osx64 -lbundle -lluajit
#cgo windows,386 LDFLAGS: -L ${SRCDIR}/lib/mingw32 -lbundle -lluajit -lmingwex -lmingw32
#cgo windows,amd64 LDFLAGS: -L ${SRCDIR}/lib/mingw64 -lbundle -lluajit -lmingwex -lmingw32

*/
import "C"
