package lib

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	ModuleVersionSeparator   = "@"
	ModuleNamespaceSeparator = ":"
	ModuleNameRegex          = `[a-z0-9]+(-[a-z0-9]+)*`
	ModuleVersionRegex       = `[0-9]+\.[0-9]+\.[0-9]+`
	ModulePreversionRegex    = `(-[a-z0-9]+)*`
)

var (
	ModuleNameRegexp    = regexp.MustCompile(fmt.Sprintf(`^%s$`, ModuleNameRegex))
	ModuleVersionRegexp = regexp.MustCompile(fmt.Sprintf(`^%s$`, ModuleVersionRegex))
)

var ModuleStringRegexp = regexp.MustCompile(fmt.Sprintf(
	`^(?P<namespace>%[1]s)%[4]s(?P<module>%[1]s)%[5]sv(?P<version>%[2]s)(?P<preversion>%[3]s)$`,
	ModuleNameRegex, ModuleVersionRegex, ModulePreversionRegex,
	ModuleNamespaceSeparator, ModuleVersionSeparator,
))

type DecodedModule struct {
	Namespace  string `json:"namespace"`
	Module     string `json:"module"`
	Version    string `json:"version"`
	Preversion string `json:"preversion"`
}

func (m DecodedModule) String() string {
	str := fmt.Sprintf(
		"%s%s%s",
		m.Namespace,
		ModuleNamespaceSeparator,
		m.Module,
	)

	if m.Version != "" {
		str += fmt.Sprintf("%sv%s%s", ModuleVersionSeparator, m.Version, m.Preversion)
	}

	return str
}

func DecodeModule(module string) DecodedModule {
	matches := ModuleStringRegexp.FindStringSubmatch(module)
	result := DecodedModule{}

	for i, name := range ModuleStringRegexp.SubexpNames() {
		if i == 0 || len(matches) <= i {
			continue
		}

		switch name {
		case "namespace":
			result.Namespace = matches[i]
		case "module":
			result.Module = matches[i]
		case "version":
			result.Version = matches[i]
		case "preversion":
			result.Preversion = matches[i]
		}
	}

	return result
}

// VersionlessModule returns the version-less representation of a module string.
func VersionlessModule(module string) string {
	parts := strings.Split(module, ModuleVersionSeparator)
	if len(parts) == 0 {
		return ""
	}

	return parts[0]
}

func isVersionlessModule(module string) bool {
	// Check if the version separator is present.
	return !strings.Contains(module, ModuleVersionSeparator)
}

// CompareModules compares two module strings.
//
// The expected value may be a version-less string (omitting the `@v...` part).
// In this case, the value will match regardless of its version.
func CompareModules(expect, value string) bool {
	if isVersionlessModule(expect) {
		return VersionlessModule(value) == expect
	}

	return value == expect
}
