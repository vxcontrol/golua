package lua

/*
#cgo CFLAGS: -I ${SRCDIR} -I ${SRCDIR}/lua

#include "golua.h"

*/
import "C"

import "unsafe"

// lua_checkstack
func (L *State) CheckStack(extra int) bool {
	defer L.r.Unlock()
	L.r.Lock()
	return C.lua_checkstack(L.s, C.int(extra)) != 0
}

// lua_pop
func (L *State) Pop(n int) {
	//Why is this implemented this way? I don't get it... maybe it
	// is just inlining manually the actual implementation.
	//C.lua_pop(L.s, C.int(n))
	defer L.r.Unlock()
	L.r.Lock()
	C.lua_settop(L.s, C.int(-n-1))
}

// lua_gettop
func (L *State) GetTop() int {
	defer L.r.Unlock()
	L.r.Lock()
	return int(C.lua_gettop(L.s))
}

// lua_settop
func (L *State) SetTop(index int) {
	defer L.r.Unlock()
	L.r.Lock()
	C.lua_settop(L.s, C.int(index))
}

// Pushes on the stack the value of a global variable (lua_getglobal)
func (L *State) GetGlobal(name string) {
	defer L.r.Unlock()
	L.r.Lock()
	L.GetField(LUA_GLOBALSINDEX, name)
}

// lua_setglobal
func (L *State) SetGlobal(name string) {
	Cname := C.CString(name)
	defer C.free(unsafe.Pointer(Cname))
	defer L.r.Unlock()
	L.r.Lock()
	C.lua_setfield(L.s, C.int(LUA_GLOBALSINDEX), Cname)
}

// lua_insert
func (L *State) Insert(index int) {
	defer L.r.Unlock()
	L.r.Lock()
	C.lua_insert(L.s, C.int(index))
}

// lua_remove
func (L *State) Remove(index int) {
	defer L.r.Unlock()
	L.r.Lock()
	C.lua_remove(L.s, C.int(index))
}

// lua_replace
func (L *State) Replace(index int) {
	defer L.r.Unlock()
	L.r.Lock()
	C.lua_replace(L.s, C.int(index))
}

// lua_rawgeti
func (L *State) RawGeti(index int, n int) {
	defer L.r.Unlock()
	L.r.Lock()
	C.lua_rawgeti(L.s, C.int(index), C.int(n))
}

// lua_rawseti
func (L *State) RawSeti(index int, n int) {
	defer L.r.Unlock()
	L.r.Lock()
	C.lua_rawseti(L.s, C.int(index), C.int(n))
}

// lua_createtable
func (L *State) CreateTable(narr int, nrec int) {
	defer L.r.Unlock()
	L.r.Lock()
	C.lua_createtable(L.s, C.int(narr), C.int(nrec))
}

// Like lua_pushcfunction pushes onto the stack a go function as user data
func (L *State) PushGoFunction(f LuaGoFunction) {
	defer L.r.Unlock()
	L.r.Lock()
	fid := L.register(f)
	C.clua_pushgofunction(L.s, C.uint(fid))
}

// PushGoClosure pushes a lua.LuaGoFunction to the stack wrapped in a Closure.
// this permits the go function to reflect lua type 'function' when checking with type()
// this implements behaviour akin to lua_pushcfunction() in lua C API.
func (L *State) PushGoClosure(f LuaGoFunction) {
	L.PushGoFunction(f) // leaves Go function userdata on stack
	defer L.r.Unlock()
	L.r.Lock()
	C.clua_pushcallback(L.s) // wraps the userdata object with a closure making it into a function
}

func (L *State) PushInt64(n int64) {
	defer L.r.Unlock()
	L.r.Lock()
	C.clua_luajit_push_cdata_int64(L.s, C.int64_t(n))
}

func (L *State) PushUint64(u uint64) {
	defer L.r.Unlock()
	L.r.Lock()
	C.clua_luajit_push_cdata_uint64(L.s, C.uint64_t(u))
}

// Pushes a Go struct onto the stack as user data.
//
// The user data will be rigged so that lua code can access
// and change the public members of simple types directly
func (L *State) PushGoStruct(iface interface{}) {
	defer L.r.Unlock()
	L.r.Lock()
	iid := L.register(iface)
	C.clua_pushgostruct(L.s, C.uint(iid))
}

// Push a pointer onto the stack as user data.
//
// This function doesn't save a reference to the interface,
// it is the responsibility of the caller of this function to insure
// that the interface outlasts the lifetime of the lua object that this function creates.
func (L *State) PushLightUserdata(ud *interface{}) {
	defer L.r.Unlock()
	L.r.Lock()
	C.lua_pushlightuserdata(L.s, unsafe.Pointer(ud))
}

// lua_pushstring
func (L *State) PushString(str string) {
	Cstr := C.CString(str)
	defer C.free(unsafe.Pointer(Cstr))
	defer L.r.Unlock()
	L.r.Lock()
	C.lua_pushlstring(L.s, Cstr, C.size_t(len(str)))
}

func (L *State) PushBytes(b []byte) {
	defer L.r.Unlock()
	L.r.Lock()
	C.lua_pushlstring(L.s, (*C.char)(unsafe.Pointer(&b[0])), C.size_t(len(b)))
}

// lua_pushinteger
func (L *State) PushInteger(n int64) {
	defer L.r.Unlock()
	L.r.Lock()
	C.lua_pushinteger(L.s, C.lua_Integer(n))
}

// lua_pushnil
func (L *State) PushNil() {
	defer L.r.Unlock()
	L.r.Lock()
	C.lua_pushnil(L.s)
}

// lua_pushnumber
func (L *State) PushNumber(n float64) {
	defer L.r.Unlock()
	L.r.Lock()
	C.lua_pushnumber(L.s, C.lua_Number(n)) // lua_Number is a cast
}

// lua_pushboolean
func (L *State) PushBoolean(b bool) {
	var bint int
	if b {
		bint = 1
	} else {
		bint = 0
	}
	defer L.r.Unlock()
	L.r.Lock()
	C.lua_pushboolean(L.s, C.int(bint))
}

// lua_pushthread
func (L *State) PushThread() (isMain bool) {
	defer L.r.Unlock()
	L.r.Lock()
	return C.lua_pushthread(L.s) != 0
}

// lua_pushvalue
func (L *State) PushValue(index int) {
	defer L.r.Unlock()
	L.r.Lock()
	C.lua_pushvalue(L.s, C.int(index))
}
