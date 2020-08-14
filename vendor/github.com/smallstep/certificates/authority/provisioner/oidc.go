package provisioner

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"net"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/smallstep/certificates/errs"
	"github.com/smallstep/cli/jose"
)

// openIDConfiguration contains the necessary properties in the
// `/.well-known/openid-configuration` document.
type openIDConfiguration struct {
	Issuer    string `json:"issuer"`
	JWKSetURI string `json:"jwks_uri"`
}

// Validate validates the values in a well-known OpenID configuration endpoint.
func (c openIDConfiguration) Validate() error {
	switch {
	case c.Issuer == "":
		return errors.New("issuer cannot be empty")
	case c.JWKSetURI == "":
		return errors.New("jwks_uri cannot be empty")
	default:
		return nil
	}
}

// openIDPayload represents the fields on the id_token JWT payload.
type openIDPayload struct {
	jose.Claims
	AtHash          string   `json:"at_hash"`
	AuthorizedParty string   `json:"azp"`
	Email           string   `json:"email"`
	EmailVerified   bool     `json:"email_verified"`
	Hd              string   `json:"hd"`
	Nonce           string   `json:"nonce"`
	Groups          []string `json:"groups"`
}

// OIDC represents an OAuth 2.0 OpenID Connect provider.
//
// ClientSecret is mandatory, but it can be an empty string.
type OIDC struct {
	*base
	Type                  string   `json:"type"`
	Name                  string   `json:"name"`
	ClientID              string   `json:"clientID"`
	ClientSecret          string   `json:"clientSecret"`
	ConfigurationEndpoint string   `json:"configurationEndpoint"`
	TenantID              string   `json:"tenantID,omitempty"`
	Admins                []string `json:"admins,omitempty"`
	Domains               []string `json:"domains,omitempty"`
	Groups                []string `json:"groups,omitempty"`
	ListenAddress         string   `json:"listenAddress,omitempty"`
	Claims                *Claims  `json:"claims,omitempty"`
	configuration         openIDConfiguration
	keyStore              *keyStore
	claimer               *Claimer
	getIdentityFunc       GetIdentityFunc
}

// IsAdmin returns true if the given email is in the Admins whitelist, false
// otherwise.
func (o *OIDC) IsAdmin(email string) bool {
	email = sanitizeEmail(email)
	for _, e := range o.Admins {
		if email == sanitizeEmail(e) {
			return true
		}
	}
	return false
}

func sanitizeEmail(email string) string {
	if i := strings.LastIndex(email, "@"); i >= 0 {
		email = email[:i] + strings.ToLower(email[i:])
	}
	return email
}

// GetID returns the provisioner unique identifier, the OIDC provisioner the
// uses the clientID for this.
func (o *OIDC) GetID() string {
	return o.ClientID
}

// GetTokenID returns the provisioner unique identifier, the OIDC provisioner the
// uses the clientID for this.
func (o *OIDC) GetTokenID(ott string) (string, error) {
	// Validate payload
	token, err := jose.ParseSigned(ott)
	if err != nil {
		return "", errors.Wrap(err, "error parsing token")
	}

	// Get claims w/out verification. We need to look up the provisioner
	// key in order to verify the claims and we need the issuer from the claims
	// before we can look up the provisioner.
	var claims openIDPayload
	if err = token.UnsafeClaimsWithoutVerification(&claims); err != nil {
		return "", errors.Wrap(err, "error verifying claims")
	}
	return claims.Nonce, nil
}

// GetName returns the name of the provisioner.
func (o *OIDC) GetName() string {
	return o.Name
}

// GetType returns the type of provisioner.
func (o *OIDC) GetType() Type {
	return TypeOIDC
}

// GetEncryptedKey is not available in an OIDC provisioner.
func (o *OIDC) GetEncryptedKey() (kid string, key string, ok bool) {
	return "", "", false
}

// Init validates and initializes the OIDC provider.
func (o *OIDC) Init(config Config) (err error) {
	switch {
	case o.Type == "":
		return errors.New("type cannot be empty")
	case o.Name == "":
		return errors.New("name cannot be empty")
	case o.ClientID == "":
		return errors.New("clientID cannot be empty")
	case o.ConfigurationEndpoint == "":
		return errors.New("configurationEndpoint cannot be empty")
	}

	// Validate listenAddress if given
	if o.ListenAddress != "" {
		if _, _, err := net.SplitHostPort(o.ListenAddress); err != nil {
			return errors.Wrap(err, "error parsing listenAddress")
		}
	}

	// Update claims with global ones
	if o.claimer, err = NewClaimer(o.Claims, config.Claims); err != nil {
		return err
	}

	// Decode and validate openid-configuration endpoint
	u, err := url.Parse(o.ConfigurationEndpoint)
	if err != nil {
		return errors.Wrapf(err, "error parsing %s", o.ConfigurationEndpoint)
	}
	if !strings.Contains(u.Path, "/.well-known/openid-configuration") {
		u.Path = path.Join(u.Path, "/.well-known/openid-configuration")
	}
	if err := getAndDecode(u.String(), &o.configuration); err != nil {
		return err
	}
	if err := o.configuration.Validate(); err != nil {
		return errors.Wrapf(err, "error parsing %s", o.ConfigurationEndpoint)
	}
	// Replace {tenantid} with the configured one
	if o.TenantID != "" {
		o.configuration.Issuer = strings.Replace(o.configuration.Issuer, "{tenantid}", o.TenantID, -1)
	}
	// Get JWK key set
	o.keyStore, err = newKeyStore(o.configuration.JWKSetURI)
	if err != nil {
		return err
	}

	// Set the identity getter if it exists, otherwise use the default.
	if config.GetIdentityFunc == nil {
		o.getIdentityFunc = DefaultIdentityFunc
	} else {
		o.getIdentityFunc = config.GetIdentityFunc
	}
	return nil
}

// ValidatePayload validates the given token payload.
func (o *OIDC) ValidatePayload(p openIDPayload) error {
	// According to "rfc7519 JSON Web Token" acceptable skew should be no more
	// than a few minutes.
	if err := p.ValidateWithLeeway(jose.Expected{
		Issuer:   o.configuration.Issuer,
		Audience: jose.Audience{o.ClientID},
		Time:     time.Now().UTC(),
	}, time.Minute); err != nil {
		return errs.Wrap(http.StatusUnauthorized, err, "validatePayload: failed to validate oidc token payload")
	}

	// Validate azp if present
	if p.AuthorizedParty != "" && p.AuthorizedParty != o.ClientID {
		return errs.Unauthorized("validatePayload: failed to validate oidc token payload: invalid azp")
	}

	// Enforce an email claim
	if p.Email == "" {
		return errs.Unauthorized("validatePayload: failed to validate oidc token payload: email not found")
	}

	// Validate domains (case-insensitive)
	if !o.IsAdmin(p.Email) && len(o.Domains) > 0 {
		email := sanitizeEmail(p.Email)
		var found bool
		for _, d := range o.Domains {
			if strings.HasSuffix(email, "@"+strings.ToLower(d)) {
				found = true
				break
			}
		}
		if !found {
			return errs.Unauthorized("validatePayload: failed to validate oidc token payload: email is not allowed")
		}
	}

	// Filter by oidc group claim
	if len(o.Groups) > 0 {
		var found bool
		for _, group := range o.Groups {
			for _, g := range p.Groups {
				if g == group {
					found = true
					break
				}
			}
		}
		if !found {
			return errs.Unauthorized("validatePayload: oidc token payload validation failed: invalid group")
		}
	}

	return nil
}

// authorizeToken applies the most common provisioner authorization claims,
// leaving the rest to context specific methods.
func (o *OIDC) authorizeToken(token string) (*openIDPayload, error) {
	jwt, err := jose.ParseSigned(token)
	if err != nil {
		return nil, errs.Wrap(http.StatusUnauthorized, err,
			"oidc.AuthorizeToken; error parsing oidc token")
	}

	// Parse claims to get the kid
	var claims openIDPayload
	if err := jwt.UnsafeClaimsWithoutVerification(&claims); err != nil {
		return nil, errs.Wrap(http.StatusUnauthorized, err,
			"oidc.AuthorizeToken; error parsing oidc token claims")
	}

	found := false
	kid := jwt.Headers[0].KeyID
	keys := o.keyStore.Get(kid)
	for _, key := range keys {
		if err := jwt.Claims(key, &claims); err == nil {
			found = true
			break
		}
	}
	if !found {
		return nil, errs.Unauthorized("oidc.AuthorizeToken; cannot validate oidc token")
	}

	if err := o.ValidatePayload(claims); err != nil {
		return nil, errs.Wrap(http.StatusInternalServerError, err, "oidc.AuthorizeToken")
	}

	return &claims, nil
}

// AuthorizeRevoke returns an error if the provisioner does not have rights to
// revoke the certificate with serial number in the `sub` property.
// Only tokens generated by an admin have the right to revoke a certificate.
func (o *OIDC) AuthorizeRevoke(ctx context.Context, token string) error {
	claims, err := o.authorizeToken(token)
	if err != nil {
		return errs.Wrap(http.StatusInternalServerError, err, "oidc.AuthorizeRevoke")
	}

	// Only admins can revoke certificates.
	if o.IsAdmin(claims.Email) {
		return nil
	}
	return errs.Unauthorized("oidc.AuthorizeRevoke; cannot revoke with non-admin oidc token")
}

// AuthorizeSign validates the given token.
func (o *OIDC) AuthorizeSign(ctx context.Context, token string) ([]SignOption, error) {
	claims, err := o.authorizeToken(token)
	if err != nil {
		return nil, errs.Wrap(http.StatusInternalServerError, err, "oidc.AuthorizeSign")
	}

	so := []SignOption{
		// modifiers / withOptions
		newProvisionerExtensionOption(TypeOIDC, o.Name, o.ClientID),
		profileDefaultDuration(o.claimer.DefaultTLSCertDuration()),
		// validators
		defaultPublicKeyValidator{},
		newValidityValidator(o.claimer.MinTLSCertDuration(), o.claimer.MaxTLSCertDuration()),
	}
	// Admins should be able to authorize any SAN
	if o.IsAdmin(claims.Email) {
		return so, nil
	}

	return append(so, emailOnlyIdentity(claims.Email)), nil
}

// AuthorizeRenew returns an error if the renewal is disabled.
// NOTE: This method does not actually validate the certificate or check it's
// revocation status. Just confirms that the provisioner that created the
// certificate was configured to allow renewals.
func (o *OIDC) AuthorizeRenew(ctx context.Context, cert *x509.Certificate) error {
	if o.claimer.IsDisableRenewal() {
		return errs.Unauthorized("oidc.AuthorizeRenew; renew is disabled for oidc provisioner %s", o.GetID())
	}
	return nil
}

// AuthorizeSSHSign returns the list of SignOption for a SignSSH request.
func (o *OIDC) AuthorizeSSHSign(ctx context.Context, token string) ([]SignOption, error) {
	if !o.claimer.IsSSHCAEnabled() {
		return nil, errs.Unauthorized("oidc.AuthorizeSSHSign; sshCA is disabled for oidc provisioner %s", o.GetID())
	}
	claims, err := o.authorizeToken(token)
	if err != nil {
		return nil, errs.Wrap(http.StatusInternalServerError, err, "oidc.AuthorizeSSHSign")
	}
	signOptions := []SignOption{
		// set the key id to the token email
		sshCertKeyIDModifier(claims.Email),
	}

	// Get the identity using either the default identityFunc or one injected
	// externally.
	iden, err := o.getIdentityFunc(ctx, o, claims.Email)
	if err != nil {
		return nil, errs.Wrap(http.StatusInternalServerError, err, "oidc.AuthorizeSSHSign")
	}
	defaults := SSHOptions{
		CertType:   SSHUserCert,
		Principals: iden.Usernames,
	}

	// Admin users can use any principal, and can sign user and host certificates.
	// Non-admin users can only use principals returned by the identityFunc, and
	// can only sign user certificates.
	if !o.IsAdmin(claims.Email) {
		signOptions = append(signOptions, sshCertOptionsValidator(defaults))
	}

	// Default to a user certificate with usernames as principals if those options
	// are not set.
	signOptions = append(signOptions, sshCertDefaultsModifier(defaults))

	return append(signOptions,
		// Set the default extensions
		&sshDefaultExtensionModifier{},
		// Set the validity bounds if not set.
		&sshDefaultDuration{o.claimer},
		// Validate public key
		&sshDefaultPublicKeyValidator{},
		// Validate the validity period.
		&sshCertValidityValidator{o.claimer},
		// Require all the fields in the SSH certificate
		&sshCertDefaultValidator{},
	), nil
}

// AuthorizeSSHRevoke returns nil if the token is valid, false otherwise.
func (o *OIDC) AuthorizeSSHRevoke(ctx context.Context, token string) error {
	claims, err := o.authorizeToken(token)
	if err != nil {
		return errs.Wrap(http.StatusInternalServerError, err, "oidc.AuthorizeSSHRevoke")
	}

	// Only admins can revoke certificates.
	if !o.IsAdmin(claims.Email) {
		return errs.Unauthorized("oidc.AuthorizeSSHRevoke; cannot revoke with non-admin oidc token")
	}
	return nil
}

func getAndDecode(uri string, v interface{}) error {
	resp, err := http.Get(uri)
	if err != nil {
		return errors.Wrapf(err, "failed to connect to %s", uri)
	}
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		return errors.Wrapf(err, "error reading %s", uri)
	}
	return nil
}
