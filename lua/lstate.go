package lua

/*
#cgo CFLAGS: -I ${SRCDIR} -I ${SRCDIR}/lua

#include "golua.h"

*/
import "C"
import (
	"unsafe"

	"github.com/vxcontrol/rmx"
)

// luaL_newstate
func NewState() *State {
	ls := (C.luaL_newstate())
	if ls == nil {
		return nil
	}
	L := newState(ls)
	return L
}

// Creates a new lua interpreter state with the given allocation function
func NewStateAlloc(f Alloc) *State {
	// jea: ../example/alloc.go panics... hmm.
	ls := C.clua_newstate(unsafe.Pointer(&f))
	L := newState(ls)
	L.allocfn = &f
	return L
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

// lua_call
func (L *State) Call(nargs, nresults int) error {
	return L.callEx(nargs, nresults)
}

// Like lua_call but panics on errors
func (L *State) MustCall(nargs, nresults int) {
	if err := L.callEx(nargs, nresults); err != nil {
		panic(err)
	}
}

func (L *State) GetState() *C.lua_State {
	defer L.r.Unlock()
	L.r.Lock()
	return L.s
}

// lua_gc
func (L *State) GC(what, data int) int {
	defer L.r.Unlock()
	L.r.Lock()
	return int(C.lua_gc(L.s, C.int(what), C.int(data)))
}

// Registers a Go function as a global variable
func (L *State) Register(name string, f LuaGoFunction) {
	L.PushGoFunction(f)
	L.SetGlobal(name)
}

// lua_close
func (L *State) Close() {
	defer L.r.Unlock()
	L.r.Lock()
	C.lua_close(L.s)
	unregisterGoState(L)
}

func newState(L *C.lua_State) *State {
	newstate := &State{
		s:          L,
		r:          &rmx.Mutex{},
		Shared:     newSharedByAllCoroutines(),
		IsMainCoro: true,
		CmainCo:    L,
		AllCoro:    make(map[int]*State),
	}
	newstate.MainCo = newstate
	defer newstate.r.Unlock()
	newstate.r.Lock()

	// only for main states, not additional coroutines:
	registerGoState(newstate) // sets Index
	newstate.Upos = int(C.clua_setgostate(L, C.size_t(newstate.Index)))

	// assert(Upos == 1)
	if newstate.Upos != 1 {
		panic("assert violated: we expected newstate.Upos for the main coro to always be at index 1: our code depends on that!")
	}
	newstate.MainCo.AllCoro[newstate.Upos] = newstate
	C.clua_initstate(L)
	return newstate
}

func (L *State) getFreeIndex() (index uint, ok bool) {
	freelen := len(L.Shared.freeIndices)
	//if there exist entries in the freelist
	if freelen > 0 {
		i := L.Shared.freeIndices[freelen-1]                       //get index
		L.Shared.freeIndices = L.Shared.freeIndices[0 : freelen-1] //'pop' index from list
		return i, true
	}
	return 0, false
}

//returns the registered function id
func (L *State) register(f interface{}) uint {
	index, ok := L.getFreeIndex()
	if ok {
		// reuse
		L.Shared.registry[index] = f
		return index
	}
	// add a new index
	L.Shared.registry = append(L.Shared.registry, f)
	index = uint(len(L.Shared.registry)) - 1
	return index
}

func (L *State) unregister(fid uint) {
	if (fid < uint(len(L.Shared.registry))) && (L.Shared.registry[fid] != nil) {
		L.Shared.registry[fid] = nil
		L.Shared.freeIndices = append(L.Shared.freeIndices, fid)
	}
}

// Sets the AtPanic function, returns the old one
//
// BUG(everyone_involved): passing nil causes serious problems
func (L *State) AtPanic(panicf LuaGoFunction) (oldpanicf LuaGoFunction) {
	defer L.r.Unlock()
	L.r.Lock()
	fid := uint(0)
	if panicf != nil {
		fid = L.register(panicf)
	}
	oldres := C.clua_atpanic(L.s, C.uint(fid))
	switch oldres.t {
	case 1:
		i := (*C.uint)(oldres.v)
		f := L.Shared.registry[uint(*i)].(LuaGoFunction)
		//free registry entry
		L.unregister(uint(*i))
		return f
	case 2:
		i := (C.lua_CFunction)(oldres.v)
		return func(L1 *State) int {
			defer L1.r.Unlock()
			L1.r.Lock()
			return int(C.clua_callluacfunc(L1.s, i))
		}
	}
	//generally we only get here if the panicf got set to something like nil
	//potentially dangerous because we may silently fail
	return nil
}

func (L *State) pcall(nargs, nresults, errfunc int) int {
	defer L.r.Unlock()
	L.r.Lock()
	return int(C.lua_pcall(L.s, C.int(nargs), C.int(nresults), C.int(errfunc)))
}

func (L *State) callEx(nargs, nresults int) (err error) {
	defer func() {
		if errRec := recover(); errRec != nil {
			if _, ok := errRec.(error); ok {
				err = errRec.(error)
			}
			return
		}
	}()

	L.GetGlobal(C.GOLUA_DEFAULT_MSGHANDLER)
	// We must record where we put the error handler in the stack otherwise it will be impossible to remove after the pcall when nresults == LUA_MULTRET
	erridx := L.GetTop() - nargs - 1
	L.Insert(erridx)
	r := L.pcall(nargs, nresults, erridx)
	L.Remove(erridx)
	if r != 0 {
		le := &LuaError{}
		if err := le.Parse(L.ToString(-1)); err != nil {
			le = le.New(L, r, L.ToString(-1))
			L.Pop(-1)
			L.PushString(le.Error())
		} else {
			le.Code = r
		}
		return le
	}
	return nil
}
