package main

import (
	"fmt"

	"github.com/vxcontrol/golua/lua"
)

func main() {
	var (
		currentPanicf lua.LuaGoFunction
		L             *lua.State
	)

	L = lua.NewState()
	defer L.Close()
	L.OpenLibs()

	newPanicf := func(L1 *lua.State) int {
		le := (&lua.LuaError{}).New(L1, 0, L1.ToString(-1))
		fmt.Println("I AM PANICKING!!!", currentPanicf, le.Msg)
		if currentPanicf != nil {
			return currentPanicf(L1)
		}

		return 1
	}
	currentPanicf = L.AtPanic(newPanicf)

	//force a panic
	test := func(L1 *lua.State) int {
		L1.RaiseError("panic check")
		return 0
	}
	L.PushGoFunction(test)
	L.Call(0, 0)

	fmt.Println("End")
}
