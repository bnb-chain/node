package types

type Script func(ctx Context, tx Msg) Error

var scriptsHub = map[string][]Script{}

func RegisterScripts(msgType string, scripts ...Script) {
	scriptsHub[msgType] = append(scriptsHub[msgType], scripts...)
}

func GetRegisteredScripts(msgType string) []Script {
	return scriptsHub[msgType]
}
