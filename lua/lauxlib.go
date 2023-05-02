package lua

/*
#cgo CFLAGS: -I ${SRCDIR} -I ${SRCDIR}/lua

#include "golua.h"

*/
import "C"

import (
	"fmt"
	"os"
	"unsafe"
)

func (L *State) Lock() {
	L.r.Lock()
}

func (L *State) Unlock() error {
	return L.r.Unlock()
}

func abs(v int) int {
	if v < 0 {
		return v * (-1)
	}
	return v
}

// lua_xmove
func XMove(from *State, to *State, n int) {
	defer from.r.Unlock()
	defer to.r.Unlock()
	from.r.Lock()
	to.r.Lock()
	C.lua_xmove(from.s, to.s, C.int(n))
}

func (L *State) CheckStackArg(narg int) {
	if int(C.lua_gettop(L.s)) < abs(narg) {
		L.RaiseError("stack dosen't contains element")
	}
}

// luaL_argcheck
// WARNING: before b30b2c62c6712c6683a9d22ff0abfa54c8267863 the function ArgCheck had the opposite behaviour
func (L *State) Argcheck(cond bool, narg int, extramsg string) {
	if !cond {
		Cextramsg := C.CString(extramsg)
		defer C.free(unsafe.Pointer(Cextramsg))
		defer L.r.Unlock()
		L.r.Lock()
		C.luaL_argerror(L.s, C.int(narg), Cextramsg)
	}
}

// luaL_argerror is dangerous on windows system
func (L *State) ArgError(narg int, extramsg string) int {
	Cextramsg := C.CString(extramsg)
	defer C.free(unsafe.Pointer(Cextramsg))
	defer L.r.Unlock()
	L.r.Lock()
	return int(C.luaL_argerror(L.s, C.int(narg), Cextramsg))
}

// luaL_argerror is dangerous on windows system
func (L *State) LuaError(msg string) int {
	defer L.r.Unlock()
	L.r.Lock()
	L.PushString(msg)
	return int(C.lua_error(L.s))
}

// luaL_callmeta
func (L *State) CallMeta(obj int, e string) int {
	Ce := C.CString(e)
	defer C.free(unsafe.Pointer(Ce))
	defer L.r.Unlock()
	L.r.Lock()
	return int(C.luaL_callmeta(L.s, C.int(obj), Ce))
}

// luaL_checkany isn't work on windows due lua_error call
func (L *State) CheckAny(narg int) {
	defer L.r.Unlock()
	L.r.Lock()
	L.CheckStackArg(narg)
}

// luaL_checkinteger isn't work on windows due lua_error call
func (L *State) CheckInteger(narg int) int {
	defer L.r.Unlock()
	L.r.Lock()
	L.CheckStackArg(narg)
	if !L.IsNumber(narg) {
		L.raiseArgumentError(narg, LUA_TNUMBER)
	}
	return L.ToInteger(narg)
}

// luaL_checknumber isn't work on windows due lua_error call
func (L *State) CheckNumber(narg int) float64 {
	defer L.r.Unlock()
	L.r.Lock()
	L.CheckStackArg(narg)
	if !L.IsNumber(narg) {
		L.raiseArgumentError(narg, LUA_TNUMBER)
	}
	return L.ToNumber(narg)
}

// luaL_checkstring isn't work on windows due lua_error call
func (L *State) CheckString(narg int) string {
	defer L.r.Unlock()
	L.r.Lock()
	L.CheckStackArg(narg)
	if !L.IsString(narg) {
		L.raiseArgumentError(narg, LUA_TSTRING)
	}
	return L.ToString(narg)
}

// luaL_checkoption
//
// BUG(everyone_involved): not implemented
func (L *State) CheckOption(narg int, def string, lst []string) int {
	//TODO: complication: lst conversion to const char* lst[] from string slice
	return 0
}

// luaL_checktype isn't work on windows due lua_error call
func (L *State) CheckType(narg int, t LuaValType) {
	defer L.r.Unlock()
	L.r.Lock()
	L.CheckStackArg(narg)
	vt := C.lua_type(L.s, C.int(narg))
	if LuaValType(vt) != t {
		L.raiseArgumentError(narg, t)
	}
}

// luaL_checkudata isn't work on windows due lua_error call
func (L *State) CheckUdata(narg int, tname string) unsafe.Pointer {
	Ctname := C.CString(tname)
	defer C.free(unsafe.Pointer(Ctname))
	defer L.r.Unlock()
	L.r.Lock()
	L.CheckStackArg(narg)
	if !L.IsUserdata(narg) {
		L.raiseArgumentError(narg, LUA_TUSERDATA)
	}
	return unsafe.Pointer(C.clua_testudata(L.s, C.int(narg), Ctname))
}

// Executes file, returns nil for no errors or the lua error string on failure
func (L *State) DoFile(filename string) error {
	if r := L.LoadFile(filename); r != 0 {
		return (&LuaError{}).New(L, r, L.ToString(-1))
	}
	return L.Call(0, LUA_MULTRET)
}

// Executes the string, returns nil for no errors or the lua error string on failure
func (L *State) DoString(str string) error {
	if r := L.LoadString(str); r != 0 {
		return (&LuaError{}).New(L, r, L.ToString(-1))
	}
	return L.Call(0, LUA_MULTRET)
}

// Like DoString but panics on error
func (L *State) MustDoString(str string) {
	if err := L.DoString(str); err != nil {
		panic(err)
	}
}

// luaL_getmetafield
func (L *State) GetMetaField(obj int, e string) bool {
	Ce := C.CString(e)
	defer C.free(unsafe.Pointer(Ce))
	defer L.r.Unlock()
	L.r.Lock()
	return C.luaL_getmetafield(L.s, C.int(obj), Ce) != 0
}

// luaL_getmetatable
func (L *State) LGetMetaTable(tname string) {
	Ctname := C.CString(tname)
	defer C.free(unsafe.Pointer(Ctname))
	defer L.r.Unlock()
	L.r.Lock()
	C.lua_getfield(L.s, LUA_REGISTRYINDEX, Ctname)
}

// luaL_gsub
func (L *State) GSub(s string, p string, r string) string {
	Cs := C.CString(s)
	Cp := C.CString(p)
	Cr := C.CString(r)
	defer func() {
		C.free(unsafe.Pointer(Cs))
		C.free(unsafe.Pointer(Cp))
		C.free(unsafe.Pointer(Cr))
	}()
	defer L.r.Unlock()
	L.r.Lock()
	return C.GoString(C.luaL_gsub(L.s, Cs, Cp, Cr))
}

// luaL_loadfile
func (L *State) LoadFile(filename string) int {
	Cfilename := C.CString(filename)
	defer C.free(unsafe.Pointer(Cfilename))
	defer L.r.Unlock()
	L.r.Lock()
	return int(C.luaL_loadfile(L.s, Cfilename))
}

// luaL_loadstring
func (L *State) LoadString(s string) int {
	Cs := C.CString(s)
	defer C.free(unsafe.Pointer(Cs))
	defer L.r.Unlock()
	L.r.Lock()
	return int(C.luaL_loadstring(L.s, Cs))
}

// lua_dump
func (L *State) Dump() int {
	ret := int(C.dump_chunk(L.s))
	return ret
}

// lua_load
func (L *State) Load(bs []byte, name string) int {
	chunk := C.CString(string(bs))
	ckname := C.CString(name)
	defer C.free(unsafe.Pointer(chunk))
	defer C.free(unsafe.Pointer(ckname))
	ret := int(C.load_chunk(L.s, chunk, C.int(len(bs)), ckname))
	if ret != 0 {
		return ret
	}
	return 0
}

// lua_newthread
func (L *State) NewThread() *State {
	defer L.r.Unlock()
	L.r.Lock()
	s := C.lua_newthread(L.s)
	return L.ToThreadHelper(s)
}

// Creates a new user data object of specified size and returns it
func (L *State) NewUserdata(size uintptr) unsafe.Pointer {
	defer L.r.Unlock()
	L.r.Lock()
	return unsafe.Pointer(C.lua_newuserdata(L.s, C.size_t(size)))
}

// lua_newtable
func (L *State) NewTable() {
	defer L.r.Unlock()
	L.r.Lock()
	C.lua_createtable(L.s, 0, 0)
}

// luaL_newmetatable
func (L *State) NewMetaTable(tname string) bool {
	Ctname := C.CString(tname)
	defer C.free(unsafe.Pointer(Ctname))
	defer L.r.Unlock()
	L.r.Lock()
	return C.luaL_newmetatable(L.s, Ctname) != 0
}

// luaL_openlibs
func (L *State) OpenLibs() {
	defer L.r.Unlock()
	L.r.Lock()
	// stop collector during initialization
	C.lua_gc(L.s, LUA_GCSTOP, 0)
	C.luaL_openlibs(L.s)
	C.lua_gc(L.s, LUA_GCRESTART, -1)
	C.clua_hide_pcall(L.s)
	// load bundle loaders
	C.bundle_add_loaders(L.s)
	// load bundle main routine and initialize args
	localArgc := C.int(len(os.Args))
	localArgv := make([]*C.char, len(os.Args))
	for i, arg := range os.Args {
		localArgv[i] = C.CString(arg)
	}
	C.bundle_main(L.s, localArgc, &localArgv[0])
}

// luaL_optinteger
func (L *State) OptInteger(narg int, d int) int {
	defer L.r.Unlock()
	L.r.Lock()
	return int(C.luaL_optinteger(L.s, C.int(narg), C.lua_Integer(d)))
}

// luaL_optnumber
func (L *State) OptNumber(narg int, d float64) float64 {
	defer L.r.Unlock()
	L.r.Lock()
	return float64(C.luaL_optnumber(L.s, C.int(narg), C.lua_Number(d)))
}

// luaL_optstring
func (L *State) OptString(narg int, d string) string {
	var length C.size_t
	Cd := C.CString(d)
	defer C.free(unsafe.Pointer(Cd))
	defer L.r.Unlock()
	L.r.Lock()
	return C.GoString(C.luaL_optlstring(L.s, C.int(narg), Cd, &length))
}

// luaL_ref
func (L *State) Ref(t int) int {
	defer L.r.Unlock()
	L.r.Lock()
	return int(C.luaL_ref(L.s, C.int(t)))
}

// luaL_typename
func (L *State) LTypename(index int) string {
	defer L.r.Unlock()
	L.r.Lock()
	return C.GoString(C.lua_typename(L.s, C.lua_type(L.s, C.int(index))))
}

// luaL_unref
func (L *State) Unref(t int, ref int) {
	defer L.r.Unlock()
	L.r.Lock()
	C.luaL_unref(L.s, C.int(t), C.int(ref))
}

// luaL_where
func (L *State) Where(lvl int) {
	defer L.r.Unlock()
	L.r.Lock()
	C.luaL_where(L.s, C.int(lvl))
}

// Sets a metamethod to execute a go function
//
// The code:
//
// 	L.LGetMetaTable(tableName)
// 	L.SetMetaMethod(methodName, function)
//
// is the logical equivalent of:
//
// 	L.LGetMetaTable(tableName)
// 	L.PushGoFunction(function)
// 	L.SetField(-2, methodName)
//
// except this wouldn't work because pushing a go function results in user data not a cfunction
func (L *State) SetMetaMethod(methodName string, f LuaGoFunction) {
	L.PushGoFunction(f) // leaves Go function userdata on stack
	defer L.r.Unlock()
	L.r.Lock()
	C.clua_pushcallback(L.s) // wraps the userdata object with a closure making it into a function
	L.SetField(-2, methodName)
}

// lua_getfenv
func (L *State) GetfEnv(index int) {
	defer L.r.Unlock()
	L.r.Lock()
	C.lua_getfenv(L.s, C.int(index))
}

// lua_setfenv
func (L *State) SetfEnv(index int) {
	defer L.r.Unlock()
	L.r.Lock()
	C.lua_setfenv(L.s, C.int(index))
}

// lua_getfield
func (L *State) GetField(index int, k string) {
	Ck := C.CString(k)
	defer C.free(unsafe.Pointer(Ck))
	defer L.r.Unlock()
	L.r.Lock()
	C.lua_getfield(L.s, C.int(index), Ck)
}

// lua_setfield
func (L *State) SetField(index int, k string) {
	Ck := C.CString(k)
	defer C.free(unsafe.Pointer(Ck))
	defer L.r.Unlock()
	L.r.Lock()
	C.lua_setfield(L.s, C.int(index), Ck)
}

// lua_gettable
func (L *State) GetTable(index int) {
	defer L.r.Unlock()
	L.r.Lock()
	C.lua_gettable(L.s, C.int(index))
}

// lua_settable
func (L *State) SetTable(index int) {
	defer L.r.Unlock()
	L.r.Lock()
	C.lua_settable(L.s, C.int(index))
}

// lua_getmetatable
func (L *State) GetMetaTable(index int) bool {
	defer L.r.Unlock()
	L.r.Lock()
	return C.lua_getmetatable(L.s, C.int(index)) != 0
}

// lua_setmetatable
func (L *State) SetMetaTable(index int) {
	defer L.r.Unlock()
	L.r.Lock()
	C.lua_setmetatable(L.s, C.int(index))
}

// lua_rawequal
func (L *State) RawEqual(index1 int, index2 int) bool {
	defer L.r.Unlock()
	L.r.Lock()
	return C.lua_rawequal(L.s, C.int(index1), C.int(index2)) != 0
}

// lua_rawget
func (L *State) RawGet(index int) {
	defer L.r.Unlock()
	L.r.Lock()
	C.lua_rawget(L.s, C.int(index))
}

// lua_rawset
func (L *State) RawSet(index int) {
	defer L.r.Unlock()
	L.r.Lock()
	C.lua_rawset(L.s, C.int(index))
}

// lua_concat
func (L *State) Concat(n int) {
	defer L.r.Unlock()
	L.r.Lock()
	C.lua_concat(L.s, C.int(n))
}

// lua_equal
func (L *State) Equal(index1, index2 int) bool {
	defer L.r.Unlock()
	L.r.Lock()
	return C.lua_equal(L.s, C.int(index1), C.int(index2)) == 1
}

// lua_lessthan
func (L *State) LessThan(index1, index2 int) bool {
	defer L.r.Unlock()
	L.r.Lock()
	return C.lua_lessthan(L.s, C.int(index1), C.int(index2)) == 1
}

// lua_objlen
//
// Returns the "length" of the value at the
// given acceptable index: for strings, this
// is the string length; for tables, this
// is the result of the length operator ('#');
// for userdata, this is the size of the block
// of memory allocated for the userdata;
// for other values, it is 0.
// jea note: Despite the misleading description
// above, in 5.1 or LuaJit, ObjLen does not call
// the metamethod __len(). In 5.2 it was split
// into len and rawlen, with len calling the __len
// metamethod.
//
func (L *State) ObjLen(index int) uint {
	defer L.r.Unlock()
	L.r.Lock()
	return uint(C.lua_objlen(L.s, C.int(index)))
}

// lua_yield
func (L *State) Yield(nresults int) int {
	defer L.r.Unlock()
	L.r.Lock()
	return int(C.lua_yield(L.s, C.int(nresults)))
}

// lua_resume
func (L *State) Resume(narg int) int {
	defer L.r.Unlock()
	L.r.Lock()
	return int(C.lua_resume(L.s, C.int(narg)))
}

// lua_next
func (L *State) Next(index int) int {
	defer L.r.Unlock()
	L.r.Lock()
	return int(C.lua_next(L.s, C.int(index)))
}

// lua_status
func (L *State) Status() int {
	defer L.r.Unlock()
	L.r.Lock()
	return int(C.lua_status(L.s))
}

// lua_setallocf
func (L *State) SetAllocf(f Alloc) {
	defer L.r.Unlock()
	L.r.Lock()
	L.allocfn = &f
	C.clua_setallocf(L.s, unsafe.Pointer(L.allocfn))
}

// Restricted library opens

// Calls luaopen_base
func (L *State) OpenBase() {
	defer L.r.Unlock()
	L.r.Lock()
	C.clua_openbase(L.s)
}

// Calls luaopen_io
func (L *State) OpenIO() {
	defer L.r.Unlock()
	L.r.Lock()
	C.clua_openio(L.s)
}

// Calls luaopen_math
func (L *State) OpenMath() {
	defer L.r.Unlock()
	L.r.Lock()
	C.clua_openmath(L.s)
}

// Calls luaopen_package
func (L *State) OpenPackage() {
	defer L.r.Unlock()
	L.r.Lock()
	C.clua_openpackage(L.s)
}

// Calls luaopen_string
func (L *State) OpenString() {
	defer L.r.Unlock()
	L.r.Lock()
	C.clua_openstring(L.s)
}

// Calls luaopen_table
func (L *State) OpenTable() {
	defer L.r.Unlock()
	L.r.Lock()
	C.clua_opentable(L.s)
}

// Calls luaopen_os
func (L *State) OpenOS() {
	defer L.r.Unlock()
	L.r.Lock()
	C.clua_openos(L.s)
}

func (L *State) raiseArgumentError(narg int, t LuaValType) {
	tn := C.GoString(C.lua_typename(L.s, C.int(t)))
	vt := C.lua_type(L.s, C.int(narg))
	vtn := C.GoString(C.lua_typename(L.s, vt))
	index := narg
	if index < 0 {
		index = L.GetTop() + narg
	}
	L.RaiseError(fmt.Sprintf("bad argument #%d (%s expected, got %s)", index, tn, vtn))
}
