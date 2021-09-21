package lua

/*
#cgo CFLAGS: -I ${SRCDIR} -I ${SRCDIR}/lua

#include "golua.h"

*/
import "C"
import (
	"unsafe"
)

// Returns true if lua_type == LUA_TBOOLEAN
func (L *State) IsBoolean(index int) bool {
	defer L.r.Unlock()
	L.r.Lock()
	return LuaValType(C.lua_type(L.s, C.int(index))) == LUA_TBOOLEAN
}

// Returns true if the value at index is a LuaGoFunction
func (L *State) IsGoFunction(index int) bool {
	defer L.r.Unlock()
	L.r.Lock()
	return C.clua_isgofunction(L.s, C.int(index)) != 0
}

// Returns true if the value at index is user data pushed with PushGoStruct
func (L *State) IsGoStruct(index int) bool {
	defer L.r.Unlock()
	L.r.Lock()
	return C.clua_isgostruct(L.s, C.int(index)) != 0
}

// Returns true if the value at index is user data pushed with PushGoFunction
func (L *State) IsFunction(index int) bool {
	defer L.r.Unlock()
	L.r.Lock()
	return LuaValType(C.lua_type(L.s, C.int(index))) == LUA_TFUNCTION
}

// Returns true if the value at index is light user data
func (L *State) IsLightUserdata(index int) bool {
	defer L.r.Unlock()
	L.r.Lock()
	return LuaValType(C.lua_type(L.s, C.int(index))) == LUA_TLIGHTUSERDATA
}

// lua_isnil
func (L *State) IsNil(index int) bool {
	defer L.r.Unlock()
	L.r.Lock()
	return LuaValType(C.lua_type(L.s, C.int(index))) == LUA_TNIL
}

// lua_isnone
func (L *State) IsNone(index int) bool {
	defer L.r.Unlock()
	L.r.Lock()
	return LuaValType(C.lua_type(L.s, C.int(index))) == LUA_TNONE
}

// lua_isnoneornil
func (L *State) IsNoneOrNil(index int) bool {
	defer L.r.Unlock()
	L.r.Lock()
	return int(C.lua_type(L.s, C.int(index))) <= 0
}

// lua_isnumber
func (L *State) IsNumber(index int) bool {
	defer L.r.Unlock()
	L.r.Lock()
	return C.lua_isnumber(L.s, C.int(index)) == 1
}

// lua_isstring
func (L *State) IsString(index int) bool {
	defer L.r.Unlock()
	L.r.Lock()
	return C.lua_isstring(L.s, C.int(index)) == 1
}

// lua_istable
func (L *State) IsTable(index int) bool {
	defer L.r.Unlock()
	L.r.Lock()
	return LuaValType(C.lua_type(L.s, C.int(index))) == LUA_TTABLE
}

// lua_isthread
func (L *State) IsThread(index int) bool {
	defer L.r.Unlock()
	L.r.Lock()
	return LuaValType(C.lua_type(L.s, C.int(index))) == LUA_TTHREAD
}

// lua_isuserdata
func (L *State) IsUserdata(index int) bool {
	defer L.r.Unlock()
	L.r.Lock()
	return C.lua_isuserdata(L.s, C.int(index)) == 1
}

// lua_tointeger
func (L *State) ToInteger(index int) int {
	defer L.r.Unlock()
	L.r.Lock()
	return int(C.lua_tointeger(L.s, C.int(index)))
}

// lua_tointeger
func (L *State) ToInteger32(index int) int32 {
	defer L.r.Unlock()
	L.r.Lock()
	return int32(C.lua_tointeger(L.s, C.int(index)))
}

// lua_tointeger
func (L *State) ToInteger64(index int) int64 {
	defer L.r.Unlock()
	L.r.Lock()
	return int64(C.lua_tointeger(L.s, C.int(index)))
}

// lua_tointeger
func (L *State) ToUInteger(index int) uint {
	defer L.r.Unlock()
	L.r.Lock()
	return uint(C.lua_tointeger(L.s, C.int(index)))
}

// lua_tointeger
func (L *State) ToUInteger32(index int) uint32 {
	defer L.r.Unlock()
	L.r.Lock()
	return uint32(C.lua_tointeger(L.s, C.int(index)))
}

// lua_tointeger
func (L *State) ToUInteger64(index int) uint64 {
	defer L.r.Unlock()
	L.r.Lock()
	return uint64(C.lua_tointeger(L.s, C.int(index)))
}

// lua_tointeger
func (L *State) ToFloat32(index int) float32 {
	defer L.r.Unlock()
	L.r.Lock()
	return float32(C.lua_tonumber(L.s, C.int(index)))
}

// lua_tointeger
func (L *State) ToFloat64(index int) float64 {
	defer L.r.Unlock()
	L.r.Lock()
	return float64(C.lua_tonumber(L.s, C.int(index)))
}

// lua_tonumber
func (L *State) ToNumber(index int) float64 {
	defer L.r.Unlock()
	L.r.Lock()
	return float64(C.lua_tonumber(L.s, C.int(index)))
}

// lua_toboolean
func (L *State) ToBoolean(index int) bool {
	defer L.r.Unlock()
	L.r.Lock()
	return C.lua_toboolean(L.s, C.int(index)) != 0
}

// Returns the value at index as a Go function (it must be something pushed with PushGoFunction)
func (L *State) ToGoFunction(index int) (f LuaGoFunction) {
	if !L.IsGoFunction(index) {
		return nil
	}
	defer L.r.Unlock()
	L.r.Lock()
	fid := C.clua_togofunction(L.s, C.int(index))
	if fid < 0 {
		return nil
	}
	return L.Shared.registry[fid].(LuaGoFunction)
}

// Returns the value at index as a Go Struct (it must be something pushed with PushGoStruct)
func (L *State) ToGoStruct(index int) (f interface{}) {
	if !L.IsGoStruct(index) {
		return nil
	}
	defer L.r.Unlock()
	L.r.Lock()
	fid := C.clua_togostruct(L.s, C.int(index))
	if fid < 0 {
		return nil
	}
	return L.Shared.registry[fid]
}

// lua_tostring
func (L *State) ToString(index int) string {
	var size C.size_t
	defer L.r.Unlock()
	L.r.Lock()
	r := C.lua_tolstring(L.s, C.int(index), &size)
	return C.GoStringN(r, C.int(size))
}

func (L *State) ToBytes(index int) []byte {
	var size C.size_t
	defer L.r.Unlock()
	L.r.Lock()
	b := C.lua_tolstring(L.s, C.int(index), &size)
	return C.GoBytes(unsafe.Pointer(b), C.int(size))
}

// lua_topointer
func (L *State) ToPointer(index int) uintptr {
	defer L.r.Unlock()
	L.r.Lock()
	return uintptr(C.lua_topointer(L.s, C.int(index)))
}

// lua_tothread
func (L *State) ToThread(index int) *State {
	defer L.r.Unlock()
	L.r.Lock()
	ptr := (*C.lua_State)(unsafe.Pointer(C.lua_tothread(L.s, C.int(index))))
	if ptr == nil {
		return nil
	}
	return L.ToThreadHelper(ptr)
}

func (L *State) ToThreadHelper(ptr *C.lua_State) *State {
	if ptr == nil {
		return nil
	}
	defer L.r.Unlock()
	L.r.Lock()
	upos := int(C.clua_dedup_coro(ptr))
	already := L.MainCo.AllCoro[upos]
	if already != nil {
		return already
	}

	newstate := &State{
		s:       ptr,
		r:       L.r,
		Shared:  L.Shared,
		MainCo:  L.MainCo,
		CmainCo: L.MainCo.s,
		Index:   -1, // not the main state/main thread.
		Upos:    upos,
	}
	// don't register non-main threads in gostates[].

	// asserts that (Upos != 1)
	if newstate.Upos == 1 {
		panic("assert violated: we expected newstate.Upos to not be 1 for any non-main thread/coroutine stated! our code in c-golua.c depends on that")

	}
	if newstate.Upos == -1 {
		panic("assert violated: we expected newstate.Upos to not be -1 for any not known coroutine!")

	}
	newstate.MainCo.AllCoro[newstate.Upos] = newstate
	return newstate
}

// lua_touserdata
func (L *State) ToUserdata(index int) unsafe.Pointer {
	defer L.r.Unlock()
	L.r.Lock()
	return unsafe.Pointer(C.lua_touserdata(L.s, C.int(index)))
}

// lua_cdata_to_int64
func (L *State) CdataToInt64(index int) int64 {
	defer L.r.Unlock()
	L.r.Lock()
	return int64(C.lua_cdata_to_int64(L.s, C.int(index)))
}

// lua_cdata_to_int32
func (L *State) CdataToInt32(index int) int32 {
	defer L.r.Unlock()
	L.r.Lock()
	return int32(C.lua_cdata_to_int32(L.s, C.int(index)))
}

// lua_cdata_to_uint64
func (L *State) CdataToUint64(index int) uint64 {
	defer L.r.Unlock()
	L.r.Lock()
	return uint64(C.lua_cdata_to_uint64(L.s, C.int(index)))
}

// LuaJIT only: return ctype of the cdata at the top of the stack.
func (L *State) LuaJITctypeID(idx int) uint32 {
	defer L.r.Unlock()
	L.r.Lock()
	res := C.clua_luajit_ctypeid(L.s, C.int(idx))
	return uint32(res)
}

// lua_type
func (L *State) Type(index int) LuaValType {
	defer L.r.Unlock()
	L.r.Lock()
	return LuaValType(C.lua_type(L.s, C.int(index)))
}

// lua_typename
func (L *State) Typename(tp int) string {
	defer L.r.Unlock()
	L.r.Lock()
	return C.GoString(C.lua_typename(L.s, C.int(tp)))
}
