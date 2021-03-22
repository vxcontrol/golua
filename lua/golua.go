package lua

/*
#cgo CFLAGS: -I ${SRCDIR}/lua

#include <lua.h>
#include <lualib.h>
#include <stdlib.h>
*/
import "C"

import (
	"fmt"
	"reflect"
	"runtime/debug"
	"sync"
	"unsafe"

	"github.com/vxcontrol/rmx"
)

// Type of allocation functions to use with NewStateAlloc
type Alloc func(ptr unsafe.Pointer, osize uint, nsize uint) unsafe.Pointer

// This is the type of go function that can be registered as lua functions
type LuaGoFunction func(L *State) int

// Wrapper to keep cgo from complaining about incomplete ptr type
//export State
type State struct {
	// Wrapped lua_State object
	s *C.lua_State

	// Mutex for all C.lua* API
	r *rmx.Mutex

	// index of this object inside the goStates array
	Index int

	Shared *SharedByAllCoroutines

	IsMainCoro bool // if true, then will be registered

	MainCo  *State       // always points to the main coroutine.
	CmainCo *C.lua_State // always points to the main coroutine's C state.

	// Upos is position in uniqArray. Upos must be 1 for
	// a main state because code in c-golua.c counts on this
	// to lookup the main coroutine from a non-main
	// coroutine. As happens naturally, that means the main
	// coroutine must be registered first, before any
	// other coroutines in that main state are
	// generated/registered.
	//
	Upos int

	// Upos -> all coroutines within a main state.
	// For non-main coroutines, AllCoro is a nil map.
	//
	// TODO: currently no hooks for garbage collection
	//  from the Lua side back to Go. So when Lua
	//  deletes a coroutine, we don't notice, and
	// it stays in our maps (uniqArray, revUniq, Lmap)
	// and on the Go side (AllCoro) forever, at the moment.
	AllCoro map[int]*State
}

type SharedByAllCoroutines struct {
	// Registry of go object that have been pushed to Lua VM
	registry []interface{}

	// Freelist for funcs indices, to allow for freeing
	freeIndices []uint
}

func newSharedByAllCoroutines() *SharedByAllCoroutines {
	return &SharedByAllCoroutines{
		registry:    make([]interface{}, 0, 8),
		freeIndices: make([]uint, 0, 8),
	}
}

var goStates map[int]*State
var goStatesMutex sync.Mutex

func init() {
	goStates = make(map[int]*State, 16)
}

var nextGoStateIndex int = 1

func registerGoState(L *State) {
	goStatesMutex.Lock()
	defer goStatesMutex.Unlock()

	// This is dangerous:
	//   L.Index = uintptr(unsafe.Pointer(L))
	// Why?
	// If the Go garbage
	// collector ever does become a moving
	// collector (and the Go team has reserved
	// the right to make that happen), and
	// it just happens to swap
	// addresses of two distinct L, then we
	// could get address reuse and this would
	// over-write a previous pointer, unexpectedly deleting it.
	//
	// It is much simpler and safer just to use
	// a counter that is incremented under the
	// lock we now hold. Thus:

	L.Index = nextGoStateIndex
	nextGoStateIndex++
	goStates[L.Index] = L
}

func unregisterGoState(L *State) {
	goStatesMutex.Lock()
	defer goStatesMutex.Unlock()
	if L.Index > 0 {
		delete(goStates, L.Index)
	}
}

func getGoState(gostateindex int) *State {
	goStatesMutex.Lock()
	defer goStatesMutex.Unlock()
	return goStates[gostateindex]
}

//export golua_printstack
func golua_printstack(coro *C.lua_State, mainIndex uintptr) {
	fmt.Printf("golua_printstack() top, called with coro='%p', mainIndex=%v\n", coro, mainIndex) // coro = ; mainIndex =1
	L := getGoState(int(mainIndex))
	fmt.Printf("golua_printstack, L back is: '%#v'\n", L)
	L1 := L.ToThreadHelper(coro)
	fmt.Printf("golua_printstack, L1 back is: '%#v'\n", L1)
	DumpLuaStack(L1)
	fmt.Printf("golua_printstack, done with L1 stack dump.\n")
}

//export golua_callgofunction
func golua_callgofunction(coro *C.lua_State, coro_index uintptr, mainIndex uintptr, mainThread *C.lua_State, fid uint) int {

	defer func() {
		r := recover()
		if r != nil {
			fmt.Printf("problem in golua_callgofunction, panic happened: '%v' at\n%s\n", r, string(debug.Stack()))
			panic(r) // resume panic
		}
	}()

	var L1 *State
	if coro_index == 0 {
		// lua side created goroutine, first time seen;
		// and not yet registered on the go-side.

		L := getGoState(int(mainIndex))
		if mainThread != nil && L.s != mainThread {
			panic("mainThread pointers disaggree")
		}
		L1 = L.ToThreadHelper(coro)
	} else {
		// this is the __call() for the MT_GOFUNCTION
		L1 = getGoState(int(coro_index))
	}

	if fid < 0 {
		panic(&LuaError{0, "Requested execution of an unknown function", L1.StackTrace()})
	}
	f := L1.Shared.registry[fid].(LuaGoFunction)

	return f(L1)
}

var typeOfBytes = reflect.TypeOf([]byte(nil))

//export golua_interface_newindex_callback
func golua_interface_newindex_callback(gostateindex uintptr, iid uint, field_name_cstr *C.char) int {
	L := getGoState(int(gostateindex))
	iface := L.Shared.registry[iid]
	ifacevalue := reflect.ValueOf(iface).Elem()

	field_name := C.GoString(field_name_cstr)

	fval := ifacevalue.FieldByName(field_name)

	if fval.Kind() == reflect.Ptr {
		fval = fval.Elem()
	}

	luatype := L.Type(3)

	switch fval.Kind() {
	case reflect.Bool:
		if luatype == LUA_TBOOLEAN {
			fval.SetBool(L.ToBoolean(3))
			return 1
		} else {
			L.PushString("Wrong assignment to field " + field_name)
			return -1
		}

	case reflect.Int:
		fallthrough
	case reflect.Int8:
		fallthrough
	case reflect.Int16:
		fallthrough
	case reflect.Int32:
		fallthrough
	case reflect.Int64:
		if luatype == LUA_TNUMBER {
			fval.SetInt(L.ToInteger64(3))
			return 1
		} else {
			L.PushString("Wrong assignment to field " + field_name)
			return -1
		}

	case reflect.Uint:
		fallthrough
	case reflect.Uint8:
		fallthrough
	case reflect.Uint16:
		fallthrough
	case reflect.Uint32:
		fallthrough
	case reflect.Uint64:
		if luatype == LUA_TNUMBER {
			fval.SetUint(L.ToUInteger64(3))
			return 1
		} else {
			L.PushString("Wrong assignment to field " + field_name)
			return -1
		}

	case reflect.String:
		if luatype == LUA_TSTRING {
			fval.SetString(L.ToString(3))
			return 1
		} else {
			L.PushString("Wrong assignment to field " + field_name)
			return -1
		}

	case reflect.Float32:
		fallthrough
	case reflect.Float64:
		if luatype == LUA_TNUMBER {
			fval.SetFloat(L.ToFloat64(3))
			return 1
		} else {
			L.PushString("Wrong assignment to field " + field_name)
			return -1
		}
	case reflect.Slice:
		if fval.Type() == typeOfBytes {
			if luatype == LUA_TSTRING {
				fval.SetBytes(L.ToBytes(3))
				return 1
			} else {
				L.PushString("Wrong assignment to field " + field_name)
				return -1
			}
		}
	}

	L.PushString("Unsupported type of field " + field_name + ": " + fval.Type().String())
	return -1
}

//export golua_interface_index_callback
func golua_interface_index_callback(gostateindex uintptr, iid uint, field_name *C.char) int {
	L := getGoState(int(gostateindex))
	iface := L.Shared.registry[iid]
	ifacevalue := reflect.ValueOf(iface).Elem()

	fval := ifacevalue.FieldByName(C.GoString(field_name))

	if fval.Kind() == reflect.Ptr {
		fval = fval.Elem()
	}

	switch fval.Kind() {
	case reflect.Bool:
		L.PushBoolean(fval.Bool())
		return 1

	case reflect.Int:
		fallthrough
	case reflect.Int8:
		fallthrough
	case reflect.Int16:
		fallthrough
	case reflect.Int32:
		fallthrough
	case reflect.Int64:
		L.PushInteger(fval.Int())
		return 1

	case reflect.Uint:
		fallthrough
	case reflect.Uint8:
		fallthrough
	case reflect.Uint16:
		fallthrough
	case reflect.Uint32:
		fallthrough
	case reflect.Uint64:
		L.PushInteger(int64(fval.Uint()))
		return 1

	case reflect.String:
		L.PushString(fval.String())
		return 1

	case reflect.Float32:
		fallthrough
	case reflect.Float64:
		L.PushNumber(fval.Float())
		return 1
	case reflect.Slice:
		if fval.Type() == typeOfBytes {
			L.PushBytes(fval.Bytes())
			return 1
		}
	}

	L.PushString("Unsupported type of field: " + fval.Type().String())
	return -1
}

//export golua_gchook
func golua_gchook(main_index uintptr, id uint) int {
	L := getGoState(int(main_index))
	L.unregister(id)
	return 0
}

//export golua_callpanicfunction
func golua_callpanicfunction(gostateindex uintptr, id uint) int {
	L1 := getGoState(int(gostateindex))
	f := L1.Shared.registry[id].(LuaGoFunction)
	return f(L1)
}

//export golua_idtointerface
func golua_idtointerface(id uint) interface{} {
	return id
}

//export golua_cfunctiontointerface
func golua_cfunctiontointerface(f *uintptr) interface{} {
	return f
}

//export golua_callallocf
func golua_callallocf(fp uintptr, ptr uintptr, osize uint, nsize uint) uintptr {
	return uintptr((*((*Alloc)(unsafe.Pointer(fp))))(unsafe.Pointer(ptr), osize, nsize))
}

//export go_panic_msghandler
func go_panic_msghandler(gostateindex uintptr, z *C.char) {
	L := getGoState(int(gostateindex))
	s := C.GoString(z)

	panic(&LuaError{LUA_ERRERR, s, L.StackTrace()})
}
