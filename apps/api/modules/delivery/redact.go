package delivery

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
)

var urlPath = regexp.MustCompile(`(https?://[^/\s"'<>]+)[^\s"'<>]*`)

/*
SafeErrorText renders a transport error without the URL it was talking to.

This exists because of one specific, quiet disaster. Go returns *url.Error from
every http.Client failure, and its Error() is `Get "<the full URL>": <cause>` —
so the text carries the whole endpoint, query string included. A Discord webhook
URL *contains its token in the path*, and Antenne now puts the far side's verbatim
error into a target's health detail, which is then rendered in the dashboard,
written to the application log that Journal receives, **and sent as an alert into
real Matrix rooms**. One unreachable webhook would have published its own
credential to every one of those places.

Keeping the method and the host is what makes the message still worth reading —
"Get https://discord.com: dial tcp: i/o timeout" tells an operator it was a
network failure reaching Discord, which is the whole diagnostic value. The path
and query never carried any.
*/
func SafeErrorText(err error) string {
	if err == nil {
		return ""
	}

	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		operation := urlErr.Op
		if operation == "" {
			operation = "request"
		}
		return fmt.Sprintf("%s %s: %s", operation, safeURL(urlErr.URL), SafeErrorText(urlErr.Err))
	}

	return err.Error()
}

// safeURL keeps only the parts that cannot be a secret. A URL too malformed to
// parse is reported as its scheme or nothing at all: an unparseable string is
// exactly the case where we cannot tell which part was the credential.
func safeURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return "the configured URL"
	}
	return parsed.Scheme + "://" + parsed.Host
}

// ScrubURLs keeps only the scheme and host of every URL in text. Error text stored
// before Antenne redacted it can carry a webhook URL whose path is its credential, so
// every response that shows stored error text passes it through here on the way out.
func ScrubURLs(text string) string {
	return urlPath.ReplaceAllString(text, "$1")
}

// RedactError is the error text an API response may carry: SafeErrorText, then
// ScrubURLs for a URL quoted anywhere else in the message.
func RedactError(err error) string {
	return ScrubURLs(SafeErrorText(err))
}
