package softkms

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/x509"

	"github.com/pkg/errors"
	"github.com/smallstep/certificates/kms/apiv1"
	"github.com/smallstep/cli/crypto/keys"
	"github.com/smallstep/cli/crypto/pemutil"
)

type algorithmAttributes struct {
	Type  string
	Curve string
}

// DefaultRSAKeySize is the default size for RSA keys.
const DefaultRSAKeySize = 3072

var signatureAlgorithmMapping = map[apiv1.SignatureAlgorithm]algorithmAttributes{
	apiv1.UnspecifiedSignAlgorithm: {"EC", "P-256"},
	apiv1.SHA256WithRSA:            {"RSA", ""},
	apiv1.SHA384WithRSA:            {"RSA", ""},
	apiv1.SHA512WithRSA:            {"RSA", ""},
	apiv1.SHA256WithRSAPSS:         {"RSA", ""},
	apiv1.SHA384WithRSAPSS:         {"RSA", ""},
	apiv1.SHA512WithRSAPSS:         {"RSA", ""},
	apiv1.ECDSAWithSHA256:          {"EC", "P-256"},
	apiv1.ECDSAWithSHA384:          {"EC", "P-384"},
	apiv1.ECDSAWithSHA512:          {"EC", "P-521"},
	apiv1.PureEd25519:              {"OKP", "Ed25519"},
}

// generateKey is used for testing purposes.
var generateKey = func(kty, crv string, size int) (interface{}, interface{}, error) {
	if kty == "RSA" && size == 0 {
		size = DefaultRSAKeySize
	}
	return keys.GenerateKeyPair(kty, crv, size)
}

// SoftKMS is a key manager that uses keys stored in disk.
type SoftKMS struct{}

// New returns a new SoftKMS.
func New(ctx context.Context, opts apiv1.Options) (*SoftKMS, error) {
	return &SoftKMS{}, nil
}

// Close is a noop that just returns nil.
func (k *SoftKMS) Close() error {
	return nil
}

// CreateSigner returns a new signer configured with the given signing key.
func (k *SoftKMS) CreateSigner(req *apiv1.CreateSignerRequest) (crypto.Signer, error) {
	var opts []pemutil.Options
	if req.Password != nil {
		opts = append(opts, pemutil.WithPassword(req.Password))
	}

	switch {
	case req.Signer != nil:
		return req.Signer, nil
	case len(req.SigningKeyPEM) != 0:
		v, err := pemutil.ParseKey(req.SigningKeyPEM, opts...)
		if err != nil {
			return nil, err
		}
		sig, ok := v.(crypto.Signer)
		if !ok {
			return nil, errors.New("signingKeyPEM is not a crypto.Signer")
		}
		return sig, nil
	case req.SigningKey != "":
		v, err := pemutil.Read(req.SigningKey, opts...)
		if err != nil {
			return nil, err
		}
		sig, ok := v.(crypto.Signer)
		if !ok {
			return nil, errors.New("signingKey is not a crypto.Signer")
		}
		return sig, nil
	default:
		return nil, errors.New("failed to load softKMS: please define signingKeyPEM or signingKey")
	}
}

func (k *SoftKMS) CreateKey(req *apiv1.CreateKeyRequest) (*apiv1.CreateKeyResponse, error) {
	v, ok := signatureAlgorithmMapping[req.SignatureAlgorithm]
	if !ok {
		return nil, errors.Errorf("softKMS does not support signature algorithm '%s'", req.SignatureAlgorithm)
	}

	pub, priv, err := generateKey(v.Type, v.Curve, req.Bits)
	if err != nil {
		return nil, err
	}
	signer, ok := priv.(crypto.Signer)
	if !ok {
		return nil, errors.Errorf("softKMS createKey result is not a crypto.Signer: type %T", priv)
	}

	return &apiv1.CreateKeyResponse{
		Name:       req.Name,
		PublicKey:  pub,
		PrivateKey: priv,
		CreateSignerRequest: apiv1.CreateSignerRequest{
			Signer: signer,
		},
	}, nil
}

func (k *SoftKMS) GetPublicKey(req *apiv1.GetPublicKeyRequest) (crypto.PublicKey, error) {
	v, err := pemutil.Read(req.Name)
	if err != nil {
		return nil, err
	}

	switch vv := v.(type) {
	case *x509.Certificate:
		return vv.PublicKey, nil
	case *rsa.PublicKey, *ecdsa.PublicKey, ed25519.PublicKey:
		return vv, nil
	default:
		return nil, errors.Errorf("unsupported public key type %T", v)
	}
}
