package delivery

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"net/textproto"
	"strconv"
	"strings"

	"github.com/FacileStudio/Antenne/apps/api/modules/health"
	"github.com/FacileStudio/Antenne/apps/api/modules/settings"
)

// probeSMTP connects, negotiates TLS, authenticates and quits. It never issues
// MAIL FROM (the classic spammer fingerprint, and a fast way to get greylisted)
// and never VRFY (disabled nearly everywhere, and its replies prove nothing).
// QUIT is the right ending; RSET would be noise with no transaction to reset.
//
// A connection failure is never a textproto reply, so classifySMTPError lands
// it on unreachable; a greeting the server refuses outright classifies on its
// own reply code instead.
func (s *Sender) probeSMTP(target settings.DeliveryTarget) health.Observation {
	address := net.JoinHostPort(target.Host, strconv.Itoa(target.Port))

	client, err := connectSMTP(target, address)
	if err != nil {
		return classifySMTPError(err)
	}
	defer client.Close()

	// EHLO before anything else, so the greeting carries our name and StartTLS
	// can re-issue it: RFC 3207 requires discarding the capability list learned
	// before TLS, and net/smtp only re-sends EHLO for the name it already holds.
	if err := client.Hello(smtpEHLOName); err != nil {
		return classifySMTPError(err)
	}
	if err := upgradeSMTP(client, target); err != nil {
		return classifySMTPError(err)
	}
	if err := authenticateSMTP(client, target); err != nil {
		return classifySMTPError(err)
	}
	if err := client.Quit(); err != nil {
		return classifySMTPError(err)
	}
	return probeOK("connected to " + address)
}

// upgradeSMTP takes STARTTLS when the server offers it. A server that does not
// advertise it is left in plaintext rather than refused, because that is the
// only way an internal relay with no certificate stays usable.
func upgradeSMTP(client *smtp.Client, target settings.DeliveryTarget) error {
	if target.Secure {
		return nil
	}
	if ok, _ := client.Extension("STARTTLS"); !ok {
		return nil
	}
	return client.StartTLS(&tls.Config{ServerName: target.Host})
}

// authenticateSMTP is a no-op for a relay that takes no credential, which is
// the normal shape for one reachable only from inside the network.
func authenticateSMTP(client *smtp.Client, target settings.DeliveryTarget) error {
	if target.Username == "" || target.Password == nil || *target.Password == "" {
		return nil
	}
	return client.Auth(smtp.PlainAuth("", target.Username, *target.Password, target.Host))
}

func classifySMTPError(err error) health.Observation {
	var reply *textproto.Error
	if errors.As(err, &reply) {
		return classifySMTPReply(reply.Code, reply.Msg)
	}
	return probeFail(health.KindUnreachable, SafeErrorText(err))
}

// classifySMTPReply maps a reply code to a Kind, and is a pure function because
// this mapping is the part worth testing and the part worth getting right.
//
// Only 535 proves the credential is wrong. 454 is a temporary auth failure —
// Google's throttle and Microsoft's smart lockout both answer with it — and
// treating it as a bad password is how a monitor locks the account it monitors.
func classifySMTPReply(code int, message string) health.Observation {
	detail := strings.TrimSpace(fmt.Sprintf("%d %s", code, message))

	switch code {
	case 535:
		return probeFail(health.KindDead, detail)
	case 432:
		return probeFail(health.KindRefreshable, detail)
	case 454:
		return probeRateLimited(detail, 0)
	case 421, 450, 451, 452:
		return probeFail(health.KindUnreachable, detail)
	case 530, 534, 538:
		return probeFail(health.KindDenied, detail)
	}

	switch {
	case code >= 200 && code < 400:
		return probeOK(detail)
	case code < 500:
		return probeFail(health.KindUnreachable, detail)
	}
	// Every other permanent refusal is the far side declining us, not proof the
	// password is wrong. Blaming the credential here would suspend probing on a
	// server that is merely misconfigured.
	return probeFail(health.KindDenied, detail)
}
