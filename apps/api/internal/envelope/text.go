package envelope

import (
	"regexp"
	"strings"
)

var (
	lineBreakTag = regexp.MustCompile(`(?i)<br\s*/?>`)
	paragraphEnd = regexp.MustCompile(`(?i)</p>`)
	anyTag       = regexp.MustCompile(`<[^>]*>`)
)

// HTMLToPlainText renders the small HTML subset Nook's messages use down to
// text, for the channels that cannot display markup: Discord, ntfy, and the
// text/plain half of an SMTP message.
func HTMLToPlainText(message string) string {
	text := lineBreakTag.ReplaceAllString(message, "\n")
	text = paragraphEnd.ReplaceAllString(text, "\n\n")
	text = anyTag.ReplaceAllString(text, "")
	return strings.TrimSpace(text)
}

// EscapeHTML escapes the three characters that can break out of Nook's message
// markup. Provider payloads are third-party text and go through it before being
// interpolated into a message.
func EscapeHTML(value string) string {
	replacer := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	return replacer.Replace(value)
}
