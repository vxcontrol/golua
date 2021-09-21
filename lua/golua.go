package lua

/*
#cgo CFLAGS: -I ${SRCDIR} -I ${SRCDIR}/lua

#include "golua.h"

*/
import "C"

import (
	"reflect"
	"sync"
	"unsafe"

	"github.com/vxcontrol/rmx"
)

// Type of allocation functions to use with NewStateAlloc
type Alloc func(ptr unsafe.Pointer, osize uint, nsize uint) unsafe.Pointer

// This is the type of go function that can be registered as lua functions
type LuaGoFunction func(L *State) int

// This is the type of a go function that can be used as a lua_Hook
type HookFunction func(L *State)

// The errorstring used by State.SetExecutionLimit
const ExecutionQuantumExceeded = "Lua execution quantum exceeded"

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

	// User self defined memory alloc func for the lua State
	allocfn *Alloc

	// User defined hook function
	hookFn HookFunction
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

//export golua_callgofunction
func golua_callgofunction(coro *C.lua_State, coro_index uintptr, mainIndex uintptr, mainThread *C.lua_State, fid int) int {
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
		L1.RaiseError("Requested execution of an unknown function")
		return 1
	}
	f := L1.Shared.registry[fid].(LuaGoFunction)

	return f(L1)
}

//export golua_callgohook
func golua_callgohook(gostateindex uintptr) {
	L1 := getGoState(int(gostateindex))
	if L1.hookFn != nil {
		L1.hookFn(L1)
	}
}

var typeOfBytes = reflect.TypeOf([]byte(nil))

//export golua_interface_newindex_callback
func golua_interface_newindex_callback(coro *C.lua_State, mainIndex uintptr, iid uint, field_name_cstr *C.char) int {
	L := getGoState(int(mainIndex))
	L1 := L.ToThreadHelper(coro)
	iface := L.Shared.registry[iid]
	ifacevalue := reflect.ValueOf(iface).Elem()

	field_name := C.GoString(field_name_cstr)

	fval := ifacevalue.FieldByName(field_name)

	if fval.Kind() == reflect.Ptr {
		fval = fval.Elem()
	}

	luatype := L1.Type(3)

	switch fval.Kind() {
	case reflect.Bool:
		if luatype == LUA_TBOOLEAN {
			fval.SetBool(L1.ToBoolean(3))
			return 1
		} else {
			L1.PushString("Wrong assignment to field " + field_name)
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
			fval.SetInt(L1.ToInteger64(3))
			return 1
		} else {
			L1.PushString("Wrong assignment to field " + field_name)
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
			fval.SetUint(L1.ToUInteger64(3))
			return 1
		} else {
			L1.PushString("Wrong assignment to field " + field_name)
			return -1
		}

	case reflect.String:
		if luatype == LUA_TSTRING {
			fval.SetString(L1.ToString(3))
			return 1
		} else {
			L1.PushString("Wrong assignment to field " + field_name)
			return -1
		}

	case reflect.Float32:
		fallthrough
	case reflect.Float64:
		if luatype == LUA_TNUMBER {
			fval.SetFloat(L1.ToFloat64(3))
			return 1
		} else {
			L1.PushString("Wrong assignment to field " + field_name)
			return -1
		}
	case reflect.Slice:
		if fval.Type() == typeOfBytes {
			if luatype == LUA_TSTRING {
				fval.SetBytes(L1.ToBytes(3))
				return 1
			} else {
				L1.PushString("Wrong assignment to field " + field_name)
				return -1
			}
		}
	}

	L1.PushString("Unsupported type of field " + field_name + ": " + fval.Type().String())
	return -1
}

//export golua_interface_index_callback
func golua_interface_index_callback(coro *C.lua_State, mainIndex uintptr, iid uint, field_name *C.char) int {
	L := getGoState(int(mainIndex))
	L1 := L.ToThreadHelper(coro)
	iface := L1.Shared.registry[iid]
	ifacevalue := reflect.ValueOf(iface).Elem()

	fval := ifacevalue.FieldByName(C.GoString(field_name))

	if fval.Kind() == reflect.Ptr {
		fval = fval.Elem()
	}

	switch fval.Kind() {
	case reflect.Bool:
		L1.PushBoolean(fval.Bool())
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
		L1.PushInteger(fval.Int())
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
		L1.PushInteger(int64(fval.Uint()))
		return 1

	case reflect.String:
		L1.PushString(fval.String())
		return 1

	case reflect.Float32:
		fallthrough
	case reflect.Float64:
		L1.PushNumber(fval.Float())
		return 1
	case reflect.Slice:
		if fval.Type() == typeOfBytes {
			L1.PushBytes(fval.Bytes())
			return 1
		}
	}

	L1.PushString("Unsupported type of field: " + fval.Type().String())
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

//export golua_callallocf
func golua_callallocf(fp uintptr, ptr uintptr, osize uint, nsize uint) uintptr {
	return uintptr((*((*Alloc)(unsafe.Pointer(fp))))(unsafe.Pointer(ptr), osize, nsize))
}

//export go_panic_msghandler
func go_panic_msghandler(coro *C.lua_State, mainIndex uintptr, z *C.char) {
	L := getGoState(int(mainIndex))
	L1 := L.ToThreadHelper(coro)
	L1.Pop(-1)
	L1.PushString((&LuaError{}).New(L1, LUA_ERRRUN, C.GoString(z)).String())
}

//export go_default_panic_msghandler
func go_default_panic_msghandler(coro *C.lua_State, mainIndex uintptr, z *C.char) {
	L := getGoState(int(mainIndex))
	L1 := L.ToThreadHelper(coro)
	le := &LuaError{}
	panic(le.New(L1, LUA_ERRERR, C.GoString(z)))
}
