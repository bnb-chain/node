package cli

import "fmt"


func enableFlag(flags uint64, targetFlag uint64) (uint64, error) {
	enabledFlags := flags | targetFlag
	if flags == enabledFlags {
		return 0, fmt.Errorf("flag %x has already been enabled", targetFlag)
	}
	return enabledFlags, nil
}

func disableFlag(flags uint64, targetFlag uint64) (uint64, error) {
	inv := ^targetFlag
	disabledFlags := flags & inv
	if flags == disabledFlags {
		return 0, fmt.Errorf("flag %x has already been disabled", targetFlag)
	}
	return disabledFlags, nil
}
