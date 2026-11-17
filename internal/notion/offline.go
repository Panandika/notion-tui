package notion

import (
	"errors"
	"net"
)

// IsOffline reports whether a call failed because Notion could not be
// reached (no network, DNS failure, connection refused, timeout) rather
// than because Notion answered with an error.
func IsOffline(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr)
}
