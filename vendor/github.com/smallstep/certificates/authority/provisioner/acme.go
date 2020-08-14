package provisioner

import (
	"context"
	"crypto/x509"

	"github.com/pkg/errors"
	"github.com/smallstep/certificates/errs"
)

// ACME is the acme provisioner type, an entity that can authorize the ACME
// provisioning flow.
type ACME struct {
	*base
	Type    string  `json:"type"`
	Name    string  `json:"name"`
	Claims  *Claims `json:"claims,omitempty"`
	claimer *Claimer
}

// GetID returns the provisioner unique identifier.
func (p ACME) GetID() string {
	return "acme/" + p.Name
}

// GetTokenID returns the identifier of the token.
func (p *ACME) GetTokenID(ott string) (string, error) {
	return "", errors.New("acme provisioner does not implement GetTokenID")
}

// GetName returns the name of the provisioner.
func (p *ACME) GetName() string {
	return p.Name
}

// GetType returns the type of provisioner.
func (p *ACME) GetType() Type {
	return TypeACME
}

// GetEncryptedKey returns the base provisioner encrypted key if it's defined.
func (p *ACME) GetEncryptedKey() (string, string, bool) {
	return "", "", false
}

// Init initializes and validates the fields of a JWK type.
func (p *ACME) Init(config Config) (err error) {
	switch {
	case p.Type == "":
		return errors.New("provisioner type cannot be empty")
	case p.Name == "":
		return errors.New("provisioner name cannot be empty")
	}

	// Update claims with global ones
	if p.claimer, err = NewClaimer(p.Claims, config.Claims); err != nil {
		return err
	}

	return err
}

// AuthorizeSign does not do any validation, because all validation is handled
// in the ACME protocol. This method returns a list of modifiers / constraints
// on the resulting certificate.
func (p *ACME) AuthorizeSign(ctx context.Context, token string) ([]SignOption, error) {
	return []SignOption{
		// modifiers / withOptions
		newProvisionerExtensionOption(TypeACME, p.Name, ""),
		profileDefaultDuration(p.claimer.DefaultTLSCertDuration()),
		// validators
		defaultPublicKeyValidator{},
		newValidityValidator(p.claimer.MinTLSCertDuration(), p.claimer.MaxTLSCertDuration()),
	}, nil
}

// AuthorizeRenew returns an error if the renewal is disabled.
// NOTE: This method does not actually validate the certificate or check it's
// revocation status. Just confirms that the provisioner that created the
// certificate was configured to allow renewals.
func (p *ACME) AuthorizeRenew(ctx context.Context, cert *x509.Certificate) error {
	if p.claimer.IsDisableRenewal() {
		return errs.Unauthorized("acme.AuthorizeRenew; renew is disabled for acme provisioner %s", p.GetID())
	}
	return nil
}
