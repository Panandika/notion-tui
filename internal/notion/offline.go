package notion

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/jomei/notionapi"
)

// ReadTimeout bounds one load of something to show (a page, rows, the
// workspace tree). On a network that silently drops traffic a call would
// otherwise hang for minutes before a saved copy could be shown.
const ReadTimeout = 30 * time.Second

// IsOffline reports whether a call failed because Notion could not be
// reached or is not working: no network, DNS failure, connection refused,
// timeout, or a server error (5xx). Then a copy saved earlier is worth
// showing.
//
// It is false when Notion answered with a refusal (not found, no access,
// invalid request), when the call was cancelled, and when the connection
// was rejected for its certificate: that is a problem with the setup that
// would never go away by waiting, and must be seen.
func IsOffline(err error) bool {
	if err == nil || errors.Is(err, context.Canceled) {
		return false
	}

	var (
		certErr    *tls.CertificateVerificationError
		authority  x509.UnknownAuthorityError
		invalid    x509.CertificateInvalidError
		hostname   x509.HostnameError
		apiErr     *notionapi.Error
		networkErr net.Error
	)
	switch {
	case errors.As(err, &certErr), errors.As(err, &authority), errors.As(err, &invalid), errors.As(err, &hostname):
		return false
	case errors.As(err, &apiErr):
		return apiErr.Status >= http.StatusInternalServerError
	default:
		return errors.As(err, &networkErr)
	}
}
