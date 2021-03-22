package lua

/*
#include <lua.h>
#include <lauxlib.h>
#include <lualib.h>
#include <stdlib.h>
#include "golua.h"

*/
import "C"
import "os"
import "unsafe"

type LuaError struct {
	code       int
	message    string
	stackTrace []LuaStackEntry
}

func (err *LuaError) Error() string {
	return err.message
}

func (err *LuaError) Code() int {
	return err.code
}

func (err *LuaError) StackTrace() []LuaStackEntry {
	return err.stackTrace
}

func (L *State) Lock() {
	L.r.Lock()
}

func (L *State) Unlock() {
	L.r.Unlock()
}

func abs(v int) int {
	if v < 0 {
		return v * (-1)
	}
	return v
}

func (L *State) CheckStackArg(narg int) {
	if int(C.lua_gettop(L.s)) < abs(narg) {
		format := "stack dosen't contains element"
		CFormat := C.CString(format)
		defer C.free(unsafe.Pointer(CFormat))
		C.lua_pushlstring(L.s, CFormat, C.size_t(len(format)))
		C.lua_error(L.s)
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

// luaL_argerror
func (L *State) ArgError(narg int, extramsg string) int {
	Cextramsg := C.CString(extramsg)
	defer C.free(unsafe.Pointer(Cextramsg))
	defer L.r.Unlock()
	L.r.Lock()
	return int(C.luaL_argerror(L.s, C.int(narg), Cextramsg))
}

// luaL_callmeta
func (L *State) CallMeta(obj int, e string) int {
	Ce := C.CString(e)
	defer C.free(unsafe.Pointer(Ce))
	defer L.r.Unlock()
	L.r.Lock()
	return int(C.luaL_callmeta(L.s, C.int(obj), Ce))
}

// luaL_checkany
func (L *State) CheckAny(narg int) {
	defer L.r.Unlock()
	L.r.Lock()
	L.CheckStackArg(narg)
	C.luaL_checkany(L.s, C.int(narg))
}

// luaL_checkinteger
func (L *State) CheckInteger(narg int) int {
	defer L.r.Unlock()
	L.r.Lock()
	L.CheckStackArg(narg)
	return int(C.luaL_checkinteger(L.s, C.int(narg)))
}

// luaL_checknumber
func (L *State) CheckNumber(narg int) float64 {
	defer L.r.Unlock()
	L.r.Lock()
	L.CheckStackArg(narg)
	return float64(C.luaL_checknumber(L.s, C.int(narg)))
}

// luaL_checkstring
func (L *State) CheckString(narg int) string {
	var length C.size_t
	defer L.r.Unlock()
	L.r.Lock()
	L.CheckStackArg(narg)
	return C.GoString(C.luaL_checklstring(L.s, C.int(narg), &length))
}

// luaL_checkoption
//
// BUG(everyone_involved): not implemented
func (L *State) CheckOption(narg int, def string, lst []string) int {
	//TODO: complication: lst conversion to const char* lst[] from string slice
	return 0
}

// luaL_checktype
func (L *State) CheckType(narg int, t LuaValType) {
	defer L.r.Unlock()
	L.r.Lock()
	L.CheckStackArg(narg)
	C.luaL_checktype(L.s, C.int(narg), C.int(t))
}

// luaL_checkudata
func (L *State) CheckUdata(narg int, tname string) unsafe.Pointer {
	Ctname := C.CString(tname)
	defer C.free(unsafe.Pointer(Ctname))
	defer L.r.Unlock()
	L.r.Lock()
	L.CheckStackArg(narg)
	return unsafe.Pointer(C.luaL_checkudata(L.s, C.int(narg), Ctname))
}

// Executes file, returns nil for no errors or the lua error string on failure
func (L *State) DoFile(filename string) error {
	if r := L.LoadFile(filename); r != 0 {
		return &LuaError{r, L.ToString(-1), L.StackTrace()}
	}
	return L.Call(0, LUA_MULTRET)
}

// Executes the string, returns nil for no errors or the lua error string on failure
func (L *State) DoString(str string) error {
	if r := L.LoadString(str); r != 0 {
		return &LuaError{r, L.ToString(-1), L.StackTrace()}
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

// luaL_newmetatable
func (L *State) NewMetaTable(tname string) bool {
	Ctname := C.CString(tname)
	defer C.free(unsafe.Pointer(Ctname))
	defer L.r.Unlock()
	L.r.Lock()
	return C.luaL_newmetatable(L.s, Ctname) != 0
}

// luaL_newstate
func NewState() *State {
	ls := (C.luaL_newstate())
	if ls == nil {
		return nil
	}
	L := newState(ls)
	return L
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

// luaL_loadbuffer
func (L *State) LoadBuffer(data []byte, size int, name string) int {
	Cname := C.CString(name)
	Cdata := (*C.char)(unsafe.Pointer(&data[0]))
	defer C.free(unsafe.Pointer(Cname))
	defer L.r.Unlock()
	L.r.Lock()
	return int(C.luaL_loadbuffer(L.s, Cdata, C.size_t(size), Cname))
}
