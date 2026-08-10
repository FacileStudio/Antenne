package envelope

import (
	"os/exec"
	"strings"
	"testing"
)

// The action at the repository root signs a webhook body with
// `openssl dgst -sha256 -hmac "$SECRET"` and presents the result as
// `sha256=<hex>`. Nothing links the two but this test, so the vector below is
// the contract: it is what openssl emitted for this body and this secret, and a
// change to either side has to fail here before it can fail a deploy.
const (
	actionBody      = `{"repository":"FacileStudio/Antenne","status":"success"}`
	actionSecret    = "s3cr3t"
	actionSignature = "sha256=c398937b89652ec263767e4c626905d7cebcabd667f9471c2fdf8e589e7ac205"
)

func TestTheActionsSignatureIsAcceptedAsIs(t *testing.T) {
	if !VerifyHexSignature(actionSecret, "sha256=", actionSignature, []byte(actionBody)) {
		t.Fatal("the signature the action sends is no longer accepted by the intake")
	}
}

func TestAWrongSecretOrBodyIsRefused(t *testing.T) {
	if VerifyHexSignature("other", "sha256=", actionSignature, []byte(actionBody)) {
		t.Error("a different secret must not verify")
	}
	if VerifyHexSignature(actionSecret, "sha256=", actionSignature, []byte(actionBody+" ")) {
		t.Error("a body altered by one byte must not verify")
	}
	if VerifyHexSignature(actionSecret, "", actionSignature, []byte(actionBody)) {
		t.Error("the prefix is part of what is compared")
	}
}

// TestOpenSSLStillEmitsTheVector guards the other half: the vector above was
// produced by openssl, and a version that changed its output format would break
// every caller of the action while every Go test stayed green.
func TestOpenSSLStillEmitsTheVector(t *testing.T) {
	binary, err := exec.LookPath("openssl")
	if err != nil {
		t.Skip("openssl not installed")
	}

	cmd := exec.Command(binary, "dgst", "-sha256", "-hmac", actionSecret, "-hex")
	cmd.Stdin = strings.NewReader(actionBody)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("openssl: %v", err)
	}

	// openssl prints either the bare digest or `<algo>(stdin)= <digest>`.
	printed := strings.TrimSpace(string(out))
	if index := strings.LastIndex(printed, " "); index >= 0 {
		printed = printed[index+1:]
	}
	if got := "sha256=" + printed; got != actionSignature {
		t.Fatalf("openssl now emits %q, the intake expects %q", got, actionSignature)
	}
}
