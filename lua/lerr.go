package lua

/*
#cgo CFLAGS: -I ${SRCDIR} -I ${SRCDIR}/lua

#include "golua.h"

*/
import "C"

import (
	"bytes"
	"encoding/json"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"unsafe"
)

const goRuntimeMaxDeeps int = 50

type LuaStackEntry struct {
	Name        string `json:"name"`
	Source      string `json:"source"`
	ShortSource string `json:"short_source"`
	CurrentLine int    `json:"line"`
}

type GoStackEntry struct {
	Name        string `json:"func"`
	Source      string `json:"file"`
	CurrentLine int    `json:"line"`
}

type LuaError struct {
	Code  int             `json:"code"`
	Msg   string          `json:"message"`
	LuaS  []string        `json:"lua_stack"`
	LuaST []LuaStackEntry `json:"lua_stacktrace"`
	GoST  []GoStackEntry  `json:"go_stacktrace"`
}

func (le *LuaError) Error() string {
	buffer := bytes.Buffer{}
	buffer.WriteString(fmt.Sprintf("lua error code: %d; %s\n", le.Code, le.Msg))
	buffer.WriteString("Lua current stack dump:\n")
	for _, entry := range le.LuaS {
		buffer.WriteString(entry)
	}
	buffer.WriteString("Lua error stack trace:\n")
	for _, entry := range le.LuaST {
		buffer.WriteString("  at ")
		buffer.WriteString(entry.Name)
		buffer.WriteString("( ")
		buffer.WriteString(entry.Source)
		buffer.WriteString(":")
		buffer.WriteString(strconv.Itoa(entry.CurrentLine))
		buffer.WriteString(" )\n")
	}
	buffer.WriteString("Go error stack trace:\n")
	for _, entry := range le.GoST {
		buffer.WriteString("  at ")
		buffer.WriteString(entry.Name)
		buffer.WriteString("( ")
		buffer.WriteString(entry.Source)
		buffer.WriteString(":")
		buffer.WriteString(strconv.Itoa(entry.CurrentLine))
		buffer.WriteString(" )\n")
	}
	return buffer.String()
}

func (le *LuaError) Parse(data string) error {
	err := json.Unmarshal([]byte(data), le)
	if err != nil {
		return err
	}
	return nil
}

func (le *LuaError) String() string {
	data, err := json.Marshal(le)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func (le *LuaError) New(state *State, code int, msg string) *LuaError {
	if le.Parse(msg) == nil {
		return le
	}
	le = &LuaError{
		Code:  code,
		Msg:   msg,
		LuaS:  state.LuaStack(),
		LuaST: state.LuaStackTrace(),
		GoST:  state.GoStackTrace(),
	}
	return le
}

// Sets the lua hook (lua_sethook).
// This and SetExecutionLimit are mutual exclusive
func (L *State) SetHook(f HookFunction, instrNumber int) {
	defer L.r.Unlock()
	L.r.Lock()
	L.hookFn = f
	C.clua_sethook(L.s, C.int(instrNumber))
}

// Sets the maximum number of operations to execute at instrNumber, after this the execution ends
// This and SetHook are mutual exclusive
func (L *State) SetExecutionLimit(instrNumber int) {
	defer L.r.Unlock()
	L.r.Lock()
	L.SetHook(func(l *State) {
		l.RaiseError(ExecutionQuantumExceeded)
	}, instrNumber)
}

// lua_error
func (L *State) RaiseError(msg string) {
	L.PushString((&LuaError{}).New(L, 0, msg).String())
	C.lua_error(L.s)
}

func (L *State) NewError(msg string) *LuaError {
	return (&LuaError{}).New(L, 0, msg)
}

// Returns the current lua stack trace
func (L *State) LuaStack() []string {
	var (
		stack []string
		top   int
	)

	top = L.GetTop()
	L.PushThread()
	L.ToThread(-1)
	L.SetTop(top)
	for i := top; i >= 1; i-- {
		stack = append(stack, L.LuaStackPosToString(i))
	}
	return stack
}

// Returns the current lua stack trace
func (L *State) LuaStackTrace() []LuaStackEntry {
	r := []LuaStackEntry{}
	var d C.lua_Debug
	Sln := C.CString("Sln")
	defer C.free(unsafe.Pointer(Sln))
	defer L.r.Unlock()
	L.r.Lock()

	for depth := 0; C.lua_getstack(L.s, C.int(depth), &d) > 0; depth++ {
		C.lua_getinfo(L.s, Sln, &d)
		ssb := make([]byte, C.LUA_IDSIZE)
		for i := 0; i < C.LUA_IDSIZE; i++ {
			ssb[i] = byte(d.short_src[i])
			if ssb[i] == 0 {
				ssb = ssb[:i]
				break
			}
		}
		ss := string(ssb)

		r = append(r, LuaStackEntry{C.GoString(d.name), C.GoString(d.source), ss, int(d.currentline)})
	}

	return r
}

// Returns the current go runtime stack trace
func (L *State) GoStackTrace() []GoStackEntry {
	pc := make([]uintptr, goRuntimeMaxDeeps)
	result := make([]GoStackEntry, 0, goRuntimeMaxDeeps)
	runtime.Callers(2, pc)
	frames := runtime.CallersFrames(pc)
	skipMod := true
	for frame, ok := frames.Next(); ok; frame, ok = frames.Next() {
		if skipMod {
			if strings.HasSuffix(frame.Function, "GoStackTrace") {
				continue
			}
			if strings.HasSuffix(frame.Function, "(*LuaError).New") {
				continue
			}
			skipMod = false
		}
		result = append(result, GoStackEntry{
			Name:        frame.Function,
			Source:      frame.File,
			CurrentLine: frame.Line,
		})
	}
	return result
}

func (L *State) DumpLuaStack() {
	fmt.Printf("\n%s\n", L.DumpLuaStackAsString())
}

func (L *State) DumpLuaStackAsString() (s string) {
	top := L.GetTop()
	isMain := L.PushThread()
	thr := L.ToThread(-1)
	L.SetTop(top)
	s += "==begin DumpLuaStack"
	s += fmt.Sprintf(" (of coro %p/lua.State=%p; isMain=%v): top = %v\n", thr, thr.s, isMain, top)
	for i := top; i >= 1; i-- {

		t := L.Type(i)
		s += fmt.Sprintf("DumpLuaStack: i=%v, t= %v\n", i, t)
		s += L.LuaStackPosToString(i)
	}
	s += "==end of DumpLuaStack\n"
	return
}

func (L *State) LuaStackPosToString(i int) string {
	t := L.Type(i)

	switch t {
	case LUA_TNONE: // -1
		return fmt.Sprintf(" LUA_TNONE; i=%v was invalid index\n", i)
	case LUA_TNIL:
		return fmt.Sprintf(" LUA_TNIL: %v\n", nil)
	case LUA_TSTRING:
		return fmt.Sprintf(" String : \t%v\n", L.ToString(i))
	case LUA_TBOOLEAN:
		return fmt.Sprintf(" Bool : \t\t%v\n", L.ToBoolean(i))
	case LUA_TNUMBER:
		return fmt.Sprintf(" Number : \t%v\n", L.ToNumber(i))
	case LUA_TTABLE:
		return fmt.Sprintf(" Table : \n%s\n", L.dumpTableString(i))

	case 10: // LUA_TCDATA aka cdata
		ctype := L.LuaJITctypeID(i)
		switch ctype {
		case 5: //  int8
		case 6: //  uint8
		case 7: //  int16
		case 8: //  uint16
		case 9: //  int32
		case 10: //  uint32
		case 11: //  int64
			val := L.CdataToInt64(i)
			return fmt.Sprintf(" int64: '%v'\n", val)
		case 12: //  uint64
			val := L.CdataToUint64(i)
			return fmt.Sprintf(" uint64: '%v'\n", val)
		case 13: //  float32
		case 14: //  float64

		case 0: // means it wasn't a ctype
		}

	case LUA_TUSERDATA:
		return fmt.Sprintf(" Type(code %v/ LUA_TUSERDATA) : 0x%x with pointer %p\n", t, L.ToPointer(i), L.ToUserdata(i))
	case LUA_TFUNCTION:
		return fmt.Sprintf(" Type(code %v/ LUA_TFUNCTION) : 0x%x\n", t, L.ToPointer(i))
	case LUA_TTHREAD:
		return fmt.Sprintf(" Type(code %v/ LUA_TTHREAD) : 0x%x\n", t, L.ToPointer(i))
	case LUA_TLIGHTUSERDATA:
		return fmt.Sprintf(" Type(code %v/ LUA_TLIGHTUSERDATA) : 0x%x with pointer %p\n", t, L.ToPointer(i), L.ToUserdata(i))
	default:
	}
	return fmt.Sprintf(" Type(code %v) : 0x%x, no auto-print available.\n", t, L.ToPointer(i))
}

func (L *State) dumpTableString(index int) (s string) {
	// Push another reference to the table on top of the stack (so we know
	// where it is, and this function can work for negative, positive and
	// pseudo indices
	L.PushValue(index)
	// stack now contains: -1 => table
	L.PushNil()
	// stack now contains: -1 => nil; -2 => table
	for L.Next(-2) != 0 {

		// stack now contains: -1 => value; -2 => key; -3 => table
		// copy the key so that lua_tostring does not modify the original
		L.PushValue(-2)
		// stack now contains: -1 => key; -2 => value; -3 => key; -4 => table
		key := L.ToString(-1)
		value := L.ToString(-2)
		s += fmt.Sprintf("'%s' => '%s'\n", key, value)
		// pop value + copy of key, leaving original key
		L.Pop(2)
		// stack now contains: -1 => key; -2 => table
	}
	// stack now contains: -1 => table (when lua_next returns 0 it pops the key
	// but does not push anything.)
	// Pop table
	L.Pop(1)
	// Stack is now the same as it was on entry to this function
	return
}
