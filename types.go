package main

import "fmt"

// ScriptType of what type of script it is
type ScriptType int

// Startup / Shutdown constant for that of a startup script
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

// ParseScriptType will take the type of script string and map to the ScriptType constant
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

// Startup will return true if it's time for a startup script
func (t ScriptType) Startup() bool {
	if t == Startup {
		return true
	}
	return false
}

// Shutdown will return true if it's shutdown script time
func (t ScriptType) Shutdown() bool {
	if t == Shutdown {
		return true
	}
	return false
}
