// Package auth provides the authentication api for the opampcommander application
package auth

import "time"

// AuthnTokenResponse defines the response structure for authentication token requests.
type AuthnTokenResponse struct {
	// Token is the authentication token.
	Token string `json:"token"`
} // @name AuthnTokenResponse

// OAuth2AuthCodeURLResponse defines the response structure for OAuth2 authorization URL requests.
// It contains the URL that the client should redirect to for OAuth2 authentication.
type OAuth2AuthCodeURLResponse struct {
	// URL is the OAuth2 authorization URL.
	URL string `json:"url"`
} // @name OAuth2AuthCodeURLResponse

// DeviceAuthnTokenResponse defines the response structure for device authentication token requests.
// It is same to `oauth2.DeviceAuthResponse`.
type DeviceAuthnTokenResponse struct {
	// DeviceCode
	DeviceCode string `json:"deviceCode"`
	// UserCode is the code the user should enter at the verification uri
	UserCode string `json:"userCode"`
	// VerificationURI is where user should enter the user code
	VerificationURI string `json:"verificationUri"`
	// VerificationURIComplete (if populated) includes the user code in the verification URI.
	// This is typically shown to the user in non-textual form, such as a QR code.
	VerificationURIComplete string `json:"verificationUriComplete,omitempty"`
	// Expiry is when the device code and user code expire
	Expiry time.Time `json:"expiry,omitempty"`
	// Interval is the duration in seconds that Poll should wait between requests
	Interval int64 `json:"interval,omitempty"`
} // @name DeviceAuthnTokenResponse
