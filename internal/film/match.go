package film

import (
	"regexp"
	"strings"
)

// match is what a lookup aims at for a text of the scenario: the text itself
// or, when it holds spaces, a pattern where each space stands for any run of
// white space on the page. French labels write "ex : Natation" with a
// narrow no-break space (U+202F) a scenario types as a plain one; the
// pattern runs in the browser, whose \s covers every kind of space. exact
// anchors it to the whole text, otherwise it is a case-insensitive
// substring, as Playwright does for a plain text.
func match(text string, exact bool) any {
	words := strings.Fields(text)
	if len(words) < 2 {
		return text
	}
	for i, w := range words {
		// The pattern travels as /source/: a slash would end it.
		words[i] = strings.ReplaceAll(regexp.QuoteMeta(w), "/", `\/`)
	}
	p := strings.Join(words, `\s+`)
	if exact {
		return regexp.MustCompile(`^\s*` + p + `\s*$`)
	}
	return regexp.MustCompile(`(?i)` + p)
}

// sameSpacing folds every run of white space, no-break spaces included, to
// one plain space: how option labels are compared on the Go side.
func sameSpacing(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
