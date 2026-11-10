// Package avatar is how a person's picture reaches every surface that shows one.
//
// Antenne never copies the bytes. users.picture_url holds the identity
// provider's stable URL, written at every sign-in and backfilled at boot, and
// every response that names a person resolves it through ByEmail. Anyone
// without one is drawn from their initial by muse's Avatar.
package avatar

// ByEmail is the SQL expression for the picture of the user whose address is
// emailSQL, or ” when no user of this instance has that address.
//
// It is one expression rather than a join so a surface adds a column to its
// select and nothing else, and it compares in lower case because users.email
// is stored lowered while a member identifier or an event actor may not be.
func ByEmail(emailSQL string) string {
	return "COALESCE((SELECT users.picture_url FROM users WHERE users.email = lower(" + emailSQL + ")), '')"
}
