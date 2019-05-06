package main

import "fmt"

type ScriptType int

const (
	Startup ScriptType = iota + 1
	Shutdown
)

func (t ScriptType) String() string {
	types := [...]string{
		"startup",
		"shutdown",
	}

	if t < Startup || t > Shutdown {
		return "unknown"
	}

	return types[t-1]
}

func ParseScriptType(s string) (ScriptType, error) {
	var ret ScriptType
	switch s {
	case "startup":
		ret = Startup
		break
	case "shutdown":
		ret = Shutdown
		break
	}

	// unknown script type
	if ret > 0 {
		return ret, nil
	}

	return 0, fmt.Errorf("unknown script type")

}

func (t ScriptType) Startup() bool {
	if t == Startup {
		return true
	}
	return false
}

func (t ScriptType) Shutdown() bool {
	if t == Shutdown {
		return true
	}
	return false
}
