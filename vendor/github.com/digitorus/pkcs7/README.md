# pkcs7

[![Go Reference](https://pkg.go.dev/badge/github.com/digitorus/pkcs7.svg)](https://pkg.go.dev/github.com/digitorus/pkcs7)
[![CI](https://github.com/digitorus/pkcs7/actions/workflows/ci.yml/badge.svg?branch=master)](https://github.com/digitorus/pkcs7/actions/workflows/ci.yml)

pkcs7 implements parsing and creating signed and enveloped messages.

The module requires Go 1.27 or later.

```go
import "github.com/digitorus/pkcs7"
```

## Post-Quantum Cryptography (PQC)

CMS SignedData supports ML-DSA-44, ML-DSA-65, and ML-DSA-87 using Go's
`crypto/mldsa` and `crypto/x509` packages. Signing and verification follow
[RFC 9882](https://www.rfc-editor.org/rfc/rfc9882.html), including pure-mode
signatures, an empty ML-DSA context, SHA-512 CMS digest identifiers, and absent
ML-DSA AlgorithmIdentifier parameters.

CMS EnvelopedData supports ML-KEM-768 and ML-KEM-1024 recipients using Go's
`crypto/mlkem` package. The implementation uses the `OtherRecipientInfo` and
`KEMRecipientInfo` architecture from
[RFC 9629](https://www.rfc-editor.org/rfc/rfc9629.html), the ML-KEM conventions
from [RFC 9936](https://www.rfc-editor.org/rfc/rfc9936.html), and ML-KEM public
keys in X.509 certificates encoded according to
[RFC 9935](https://www.rfc-editor.org/rfc/rfc9935.html). It derives the
key-encryption key with HKDF-SHA256 and wraps the content-encryption key with
AES-Wrap-256. Select an AES-CBC content-encryption algorithm before calling
`Encrypt`; for example:

```go
pkcs7.ContentEncryptionAlgorithm = pkcs7.EncryptionAlgorithmAES256CBC
encrypted, err := pkcs7.Encrypt(content, recipients)
```

AES-GCM is standardized for CMS `AuthEnvelopedData` by
[RFC 5084](https://www.rfc-editor.org/rfc/rfc5084.html), not ordinary
`EnvelopedData`. This package does not currently implement
`AuthEnvelopedData`, so ML-KEM recipients reject AES-GCM content encryption.

ML-KEM-512 is standardized but is not supported because Go 1.27's standard
library provides only ML-KEM-768 and ML-KEM-1024. This package does not yet
implement hybrid or composite KEM recipients. EnvelopedData encryption alone
does not authenticate the originator; sign content when origin authentication
is required.

SLH-DSA is standardized for CMS by
[RFC 9814](https://www.rfc-editor.org/rfc/rfc9814.html), but is not supported
because Go's standard library does not currently provide an SLH-DSA
implementation.

Future work includes ML-KEM-512 when an appropriate Go implementation is
available, CMS `AuthEnvelopedData`, CMS signed-attribute and EUF-CMA hardening,
CMSAlgorithmProtection, SLH-DSA when an appropriate Go implementation is
available, and composite ML-DSA or ML-KEM only after the relevant IETF
specifications stabilize.

## Interoperability

CI exercises CMS in both directions with OpenSSL 3.6 and 4.0 and Bouncy Castle
1.85. SignedData coverage includes RSA, ECDSA, Ed25519, and all three ML-DSA
parameter sets with embedded and detached content, with and without signed
attributes where the external implementation supports those combinations.
EnvelopedData coverage includes RSA key transport with AES-128-CBC and
AES-256-CBC against OpenSSL, plus ML-KEM-768 and ML-KEM-1024 KEMRecipientInfo
with AES-256-CBC against both OpenSSL and Bouncy Castle.

```go
package main

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"

	"github.com/digitorus/pkcs7"
)

func SignAndDetach(content []byte, cert *x509.Certificate, privkey *rsa.PrivateKey) (signed []byte, err error) {
	toBeSigned, err := pkcs7.NewSignedData(content)
	if err != nil {
		err = fmt.Errorf("Cannot initialize signed data: %s", err)
		return
	}
	if err = toBeSigned.AddSigner(cert, privkey, pkcs7.SignerInfoConfig{}); err != nil {
		err = fmt.Errorf("Cannot add signer: %s", err)
		return
	}

	// Detach signature, omit if you want an embedded signature
	toBeSigned.Detach()

	signed, err = toBeSigned.Finish()
	if err != nil {
		err = fmt.Errorf("Cannot finish signing data: %s", err)
		return
	}

	// Verify the signature
	pem.Encode(os.Stdout, &pem.Block{Type: "PKCS7", Bytes: signed})
	p7, err := pkcs7.Parse(signed)
	if err != nil {
		err = fmt.Errorf("Cannot parse our signed data: %s", err)
		return
	}

	// since the signature was detached, reattach the content here
	p7.Content = content

	if bytes.Compare(content, p7.Content) != 0 {
		err = fmt.Errorf("Our content was not in the parsed data:\n\tExpected: %s\n\tActual: %s", content, p7.Content)
		return
	}
	if err = p7.Verify(); err != nil {
		err = fmt.Errorf("Cannot verify our signed data: %s", err)
		return
	}

	return signed, nil
}
```



## Credits
This is a fork of [fullsailor/pkcs7](https://github.com/fullsailor/pkcs7)
