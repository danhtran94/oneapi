package apidoc

import (
	"regexp"
	"strings"
)

type OAPIDef map[string]string

const (
	DefKind      = "kind"
	KindResponse = "response"

	ResponsePlaceholder = "placeholder"
	ResponseName        = "name"
)

func (def OAPIDef) Get(key string) string {

	if val, ok := def[key]; ok {
		return val
	}

	return ""
}

func getOAPIDef(comment string) OAPIDef {
	defs := map[string]string{}

	re := regexp.MustCompile(`(?m)@oas:.*`)
	matches := re.FindAllString(comment, -1)

	if len(matches) > 0 {
		lastLine := matches[len(matches)-1]
		resp := strings.Split(lastLine, "@oas:")[1]

		for _, def := range strings.Split(resp, " ") {
			if strings.Contains(def, "=") {
				strs := strings.Split(def, "=")
				key, value := strs[0], strs[1]
				defs[key] = value
			}
		}
	}

	return defs
}
