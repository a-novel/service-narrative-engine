package modules

import "embed"

//go:embed agora/*.yaml
var AgoraModules embed.FS

var KnownModules = map[string]embed.FS{
	"agora": AgoraModules,
}
