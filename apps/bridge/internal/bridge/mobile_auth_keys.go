package bridge

import "strings"

func (c MobileEnrollmentConfig) hasOAuthProvisioner() bool {
	return strings.TrimSpace(c.OAuthClientID) != "" &&
		strings.TrimSpace(c.OAuthClientSecret) != "" &&
		strings.TrimSpace(c.OAuthTailnet) != "" &&
		len(c.OAuthTags) > 0
}
