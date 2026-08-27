package pkcs7

import (
	"bytes"
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/des"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/asn1"
	"errors"
	"fmt"
)

// ErrUnsupportedAlgorithm tells you when our quick dev assumptions have failed
var ErrUnsupportedAlgorithm = errors.New("pkcs7: cannot decrypt data: only RSA, DES, DES-EDE3, AES-256-CBC and AES-128-GCM supported")

// ErrNotEncryptedContent is returned when attempting to Decrypt data that is not encrypted data
var ErrNotEncryptedContent = errors.New("pkcs7: content data is a decryptable data type")

// Decrypt decrypts encrypted content info for a recipient certificate and an
// RSA private key or ML-KEM-768/1024 decapsulation key.
func (p7 *PKCS7) Decrypt(cert *x509.Certificate, pkey crypto.PrivateKey) ([]byte, error) {
	data, ok := p7.raw.(envelopedData)
	if !ok {
		return nil, ErrNotEncryptedContent
	}
	switch pkey := pkey.(type) {
	case *rsa.PrivateKey:
		recipient, err := selectRecipientForCertificate(data.RecipientInfos, cert)
		if err != nil {
			return nil, err
		}
		var contentKey []byte
		//nolint:staticcheck // PKCS #1 v1.5 is required by the existing CMS RSA key-transport format.
		contentKey, err = rsa.DecryptPKCS1v15(rand.Reader, pkey, recipient.EncryptedKey)
		if err != nil {
			return nil, err
		}
		defer clear(contentKey)
		return data.EncryptedContentInfo.decrypt(contentKey)
	default:
		if _, _, ok := mlKEMDecapsulator(pkey); ok {
			if data.Version != 3 {
				return nil, errors.New("pkcs7: EnvelopedData containing OtherRecipientInfo must use version 3")
			}
			contentEncryptionAlgorithm := data.EncryptedContentInfo.ContentEncryptionAlgorithm.Algorithm
			if !contentEncryptionAlgorithm.Equal(OIDEncryptionAlgorithmAES128CBC) &&
				!contentEncryptionAlgorithm.Equal(OIDEncryptionAlgorithmAES256CBC) {
				return nil, errors.New("pkcs7: ML-KEM EnvelopedData requires AES-CBC; AES-GCM requires AuthEnvelopedData")
			}
			contentKey, err := decryptKEMRecipientKey(data.RecipientInfos, cert, pkey)
			if err != nil {
				return nil, err
			}
			defer clear(contentKey)
			return data.EncryptedContentInfo.decrypt(contentKey)
		}
	}
	return nil, ErrUnsupportedAlgorithm
}

// DecryptUsingPSK decrypts encrypted data using caller provided
// pre-shared secret
func (p7 *PKCS7) DecryptUsingPSK(key []byte) ([]byte, error) {
	data, ok := p7.raw.(encryptedData)
	if !ok {
		return nil, ErrNotEncryptedContent
	}
	return data.EncryptedContentInfo.decrypt(key)
}

func (eci encryptedContentInfo) decrypt(key []byte) ([]byte, error) {
	alg := eci.ContentEncryptionAlgorithm.Algorithm
	if !alg.Equal(OIDEncryptionAlgorithmDESCBC) &&
		!alg.Equal(OIDEncryptionAlgorithmDESEDE3CBC) &&
		!alg.Equal(OIDEncryptionAlgorithmAES256CBC) &&
		!alg.Equal(OIDEncryptionAlgorithmAES128CBC) &&
		!alg.Equal(OIDEncryptionAlgorithmAES128GCM) &&
		!alg.Equal(OIDEncryptionAlgorithmAES256GCM) {
		fmt.Printf("Unsupported Content Encryption Algorithm: %s\n", alg)
		return nil, ErrUnsupportedAlgorithm
	}

	// EncryptedContent can either be constructed of multple OCTET STRINGs
	// or _be_ a tagged OCTET STRING
	var cyphertext []byte
	if eci.EncryptedContent.IsCompound {
		// Complex case to concat all of the children OCTET STRINGs
		var buf bytes.Buffer
		cypherbytes := eci.EncryptedContent.Bytes
		for {
			var part []byte
			cypherbytes, _ = asn1.Unmarshal(cypherbytes, &part)
			buf.Write(part)
			if cypherbytes == nil {
				break
			}
		}
		cyphertext = buf.Bytes()
	} else {
		// Simple case, the bytes _are_ the cyphertext
		cyphertext = eci.EncryptedContent.Bytes
	}

	var block cipher.Block
	var err error

	switch {
	case alg.Equal(OIDEncryptionAlgorithmDESCBC):
		block, err = des.NewCipher(key)
	case alg.Equal(OIDEncryptionAlgorithmDESEDE3CBC):
		block, err = des.NewTripleDESCipher(key)
	case alg.Equal(OIDEncryptionAlgorithmAES256CBC), alg.Equal(OIDEncryptionAlgorithmAES256GCM):
		fallthrough
	case alg.Equal(OIDEncryptionAlgorithmAES128GCM), alg.Equal(OIDEncryptionAlgorithmAES128CBC):
		block, err = aes.NewCipher(key)
	}

	if err != nil {
		return nil, err
	}

	if alg.Equal(OIDEncryptionAlgorithmAES128GCM) || alg.Equal(OIDEncryptionAlgorithmAES256GCM) {
		params := aesGCMParameters{}
		paramBytes := eci.ContentEncryptionAlgorithm.Parameters.Bytes

		_, err := asn1.Unmarshal(paramBytes, &params)
		if err != nil {
			return nil, err
		}

		gcm, err := cipher.NewGCM(block)
		if err != nil {
			return nil, err
		}

		if len(params.Nonce) != gcm.NonceSize() {
			return nil, errors.New("pkcs7: encryption algorithm parameters are incorrect")
		}
		if params.ICVLen != gcm.Overhead() {
			return nil, errors.New("pkcs7: encryption algorithm parameters are incorrect")
		}

		plaintext, err := gcm.Open(nil, params.Nonce, cyphertext, nil)
		if err != nil {
			return nil, err
		}

		return plaintext, nil
	}

	iv := eci.ContentEncryptionAlgorithm.Parameters.Bytes
	if len(iv) != block.BlockSize() {
		return nil, errors.New("pkcs7: encryption algorithm parameters are malformed")
	}
	mode := cipher.NewCBCDecrypter(block, iv)
	plaintext := make([]byte, len(cyphertext))
	mode.CryptBlocks(plaintext, cyphertext)
	if plaintext, err = unpad(plaintext, mode.BlockSize()); err != nil {
		return nil, err
	}
	return plaintext, nil
}

func unpad(data []byte, blocklen int) ([]byte, error) {
	if blocklen < 1 {
		return nil, fmt.Errorf("invalid blocklen %d", blocklen)
	}
	if len(data)%blocklen != 0 || len(data) == 0 {
		return nil, fmt.Errorf("invalid data len %d", len(data))
	}

	// the last byte is the length of padding
	padlen := int(data[len(data)-1])

	// check padding integrity, all bytes should be the same
	pad := data[len(data)-padlen:]
	for _, padbyte := range pad {
		if padbyte != byte(padlen) {
			return nil, errors.New("invalid padding")
		}
	}

	return data[:len(data)-padlen], nil
}

func selectRecipientForCertificate(recipients []asn1.RawValue, cert *x509.Certificate) (recipientInfo, error) {
	if cert == nil {
		return recipientInfo{}, errors.New("pkcs7: recipient certificate is nil")
	}
	for _, raw := range recipients {
		if raw.Class != 0 || raw.Tag != asn1.TagSequence {
			continue
		}
		var recp recipientInfo
		rest, err := asn1.Unmarshal(raw.FullBytes, &recp)
		if err != nil {
			return recipientInfo{}, fmt.Errorf("pkcs7: malformed KeyTransRecipientInfo: %w", err)
		}
		if len(rest) != 0 {
			return recipientInfo{}, errors.New("pkcs7: trailing data in KeyTransRecipientInfo")
		}
		if isCertMatchForIssuerAndSerial(cert, recp.IssuerAndSerialNumber) {
			return recp, nil
		}
	}
	return recipientInfo{}, errors.New("pkcs7: no enveloped recipient for provided certificate")
}
