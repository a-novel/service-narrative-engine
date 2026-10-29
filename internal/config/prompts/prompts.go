package prompts

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"sync"
	"text/template"
)

// files holds every template of the package, read from the files beside this one at build time. The file
// context.tmpl defines the context sections, and each file <kind>.<language>.tmpl defines the developer
// message and the rules reminder of a generation kind in a language.
//
//go:embed *.tmpl
var files embed.FS

// ErrUnknownPrompt is returned by [Lookup] for a generation kind or a language without templates.
var ErrUnknownPrompt = errors.New("unknown prompt")

// escaper escapes the characters that would let text close or open an element of a prompt.
var escaper = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")

// funcs are the functions the templates call. xml escapes text written inside an element or an attribute.
var funcs = template.FuncMap{"xml": escaper.Replace}

// templates parses every template set on first use, keyed by <kind>.<language>.
var templates = sync.OnceValues(loadTemplates)

// Lookup returns the templates of a generation kind in a language, a BCP 47 tag. The set defines:
//
//   - developer, the developer message;
//   - rules, the reminder the task ends with;
//   - frame, neighborhood and task.<kind>, the sections that follow the developer message;
//   - scene and block, the parts the planner measures one at a time.
func Lookup(kind, language string) (*template.Template, error) {
	loaded, err := templates()
	if err != nil {
		return nil, err
	}

	set, ok := loaded[kind+"."+language]
	if !ok {
		return nil, fmt.Errorf("%w: %s in %s", ErrUnknownPrompt, kind, language)
	}

	return set, nil
}

// loadTemplates parses one template set per file <kind>.<language>.tmpl, each joined with context.tmpl.
func loadTemplates() (map[string]*template.Template, error) {
	names, err := fs.Glob(files, "*.*.tmpl")
	if err != nil {
		return nil, fmt.Errorf("list templates: %w", err)
	}

	loaded := make(map[string]*template.Template, len(names))

	for _, name := range names {
		set, err := template.New(name).Funcs(funcs).ParseFS(files, "context.tmpl", name)
		if err != nil {
			return nil, fmt.Errorf("parse templates %s: %w", name, err)
		}

		loaded[strings.TrimSuffix(name, ".tmpl")] = set
	}

	return loaded, nil
}
