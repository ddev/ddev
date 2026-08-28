package pkcs7

import (
	"crypto/aes"
	"crypto/hkdf"
	"crypto/mlkem"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/binary"
	"errors"
	"fmt"
)

const mlKEMKEKLength = 32

var aesKeyWrapInitialValue = [8]byte{0xa6, 0xa6, 0xa6, 0xa6, 0xa6, 0xa6, 0xa6, 0xa6}

type otherRecipientInfo struct {
	Type  asn1.ObjectIdentifier
	Value asn1.RawValue
}

type kemRecipientInfo struct {
	Version      int
	RecipientID  asn1.RawValue
	KEM          pkix.AlgorithmIdentifier
	Ciphertext   []byte
	KDF          pkix.AlgorithmIdentifier
	KEKLength    int
	UKM          []byte `asn1:"optional,explicit,tag:0"`
	Wrap         pkix.AlgorithmIdentifier
	EncryptedKey []byte
}

type cmsORIForKEMOtherInfo struct {
	Wrap      pkix.AlgorithmIdentifier
	KEKLength int
	UKM       []byte `asn1:"optional,explicit,tag:0"`
}

type subjectPublicKeyInfo struct {
	Algorithm pkix.AlgorithmIdentifier
	PublicKey asn1.BitString
}

func makeKEMRecipientInfo(contentKey []byte, recipient *x509.Certificate) (asn1.RawValue, error) {
	if ContentEncryptionAlgorithm != EncryptionAlgorithmAES128CBC && ContentEncryptionAlgorithm != EncryptionAlgorithmAES256CBC {
		return asn1.RawValue{}, errors.New("pkcs7: ML-KEM EnvelopedData requires AES-CBC; AES-GCM requires AuthEnvelopedData")
	}
	if len(contentKey) < 16 || len(contentKey)%8 != 0 {
		return asn1.RawValue{}, errors.New("pkcs7: invalid ML-KEM content-encryption key length")
	}
	if recipient == nil {
		return asn1.RawValue{}, errors.New("pkcs7: ML-KEM recipient certificate is nil")
	}
	if recipient.KeyUsage != 0 && recipient.KeyUsage != x509.KeyUsageKeyEncipherment {
		return asn1.RawValue{}, errors.New("pkcs7: ML-KEM recipient certificate key usage must contain only keyEncipherment")
	}

	kemOID, sharedSecret, ciphertext, err := encapsulateForCertificate(recipient)
	if err != nil {
		return asn1.RawValue{}, err
	}
	defer clear(sharedSecret)

	wrap := pkix.AlgorithmIdentifier{Algorithm: OIDKeyWrapAES256}
	kek, err := deriveKEMKey(sharedSecret, wrap, mlKEMKEKLength, nil)
	if err != nil {
		return asn1.RawValue{}, err
	}
	defer clear(kek)

	encryptedKey, err := aesKeyWrap(kek, contentKey)
	if err != nil {
		return asn1.RawValue{}, err
	}
	recipientID, err := recipientIdentifierForCertificate(recipient)
	if err != nil {
		return asn1.RawValue{}, err
	}

	kemInfo := kemRecipientInfo{
		Version:      0,
		RecipientID:  recipientID,
		KEM:          pkix.AlgorithmIdentifier{Algorithm: kemOID},
		Ciphertext:   ciphertext,
		KDF:          pkix.AlgorithmIdentifier{Algorithm: OIDKDFHKDFSHA256},
		KEKLength:    mlKEMKEKLength,
		Wrap:         wrap,
		EncryptedKey: encryptedKey,
	}
	kemDER, err := asn1.Marshal(kemInfo)
	if err != nil {
		return asn1.RawValue{}, err
	}
	return marshalOtherRecipientInfo(OIDOtherRecipientInfoKEM, kemDER)
}

func encapsulateForCertificate(cert *x509.Certificate) (asn1.ObjectIdentifier, []byte, []byte, error) {
	if len(cert.RawSubjectPublicKeyInfo) == 0 {
		switch publicKey := cert.PublicKey.(type) {
		case *mlkem.EncapsulationKey768:
			sharedSecret, ciphertext := publicKey.Encapsulate()
			return OIDKeyAlgorithmMLKEM768, sharedSecret, ciphertext, nil
		case *mlkem.EncapsulationKey1024:
			sharedSecret, ciphertext := publicKey.Encapsulate()
			return OIDKeyAlgorithmMLKEM1024, sharedSecret, ciphertext, nil
		default:
			return nil, nil, nil, fmt.Errorf("pkcs7: unsupported recipient public key type %T", cert.PublicKey)
		}
	}

	var spki subjectPublicKeyInfo
	rest, err := asn1.Unmarshal(cert.RawSubjectPublicKeyInfo, &spki)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("pkcs7: cannot parse ML-KEM SubjectPublicKeyInfo: %w", err)
	}
	if len(rest) != 0 {
		return nil, nil, nil, errors.New("pkcs7: trailing data in ML-KEM SubjectPublicKeyInfo")
	}
	if algorithmIdentifierHasParameters(spki.Algorithm) {
		return nil, nil, nil, errors.New("pkcs7: ML-KEM SubjectPublicKeyInfo parameters must be absent")
	}
	if spki.PublicKey.BitLength != len(spki.PublicKey.Bytes)*8 {
		return nil, nil, nil, errors.New("pkcs7: malformed ML-KEM SubjectPublicKeyInfo public key")
	}

	switch {
	case spki.Algorithm.Algorithm.Equal(OIDKeyAlgorithmMLKEM512):
		return nil, nil, nil, errors.New("pkcs7: ML-KEM-512 is not supported by Go 1.27 crypto/mlkem")
	case spki.Algorithm.Algorithm.Equal(OIDKeyAlgorithmMLKEM768):
		publicKey, err := mlkem.NewEncapsulationKey768(spki.PublicKey.Bytes)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("pkcs7: invalid ML-KEM-768 public key: %w", err)
		}
		sharedSecret, ciphertext := publicKey.Encapsulate()
		return OIDKeyAlgorithmMLKEM768, sharedSecret, ciphertext, nil
	case spki.Algorithm.Algorithm.Equal(OIDKeyAlgorithmMLKEM1024):
		publicKey, err := mlkem.NewEncapsulationKey1024(spki.PublicKey.Bytes)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("pkcs7: invalid ML-KEM-1024 public key: %w", err)
		}
		sharedSecret, ciphertext := publicKey.Encapsulate()
		return OIDKeyAlgorithmMLKEM1024, sharedSecret, ciphertext, nil
	default:
		return nil, nil, nil, fmt.Errorf("pkcs7: unsupported recipient public key algorithm %s", spki.Algorithm.Algorithm)
	}
}

func recipientIdentifierForCertificate(cert *x509.Certificate) (asn1.RawValue, error) {
	return marshalRecipientInfo(cert2issuerAndSerial(cert))
}

func marshalRecipientInfo(value any) (asn1.RawValue, error) {
	der, err := asn1.Marshal(value)
	if err != nil {
		return asn1.RawValue{}, err
	}
	var raw asn1.RawValue
	rest, err := asn1.Unmarshal(der, &raw)
	if err != nil {
		return asn1.RawValue{}, fmt.Errorf("pkcs7: cannot marshal RecipientInfo: %w", err)
	}
	if len(rest) != 0 {
		return asn1.RawValue{}, errors.New("pkcs7: trailing data in marshaled RecipientInfo")
	}
	return raw, nil
}

func marshalOtherRecipientInfo(oid asn1.ObjectIdentifier, valueDER []byte) (asn1.RawValue, error) {
	other := otherRecipientInfo{
		Type:  oid,
		Value: asn1.RawValue{FullBytes: valueDER},
	}
	sequence, err := marshalRecipientInfo(other)
	if err != nil {
		return asn1.RawValue{}, err
	}
	return asn1.RawValue{
		Class:      2,
		Tag:        4,
		IsCompound: true,
		Bytes:      sequence.Bytes,
	}, nil
}

func parseOtherRecipientInfo(raw asn1.RawValue) (otherRecipientInfo, error) {
	if raw.Class != 2 || raw.Tag != 4 || !raw.IsCompound {
		return otherRecipientInfo{}, errors.New("pkcs7: RecipientInfo is not an OtherRecipientInfo")
	}
	sequenceDER, err := asn1.Marshal(asn1.RawValue{
		Tag:        asn1.TagSequence,
		IsCompound: true,
		Bytes:      raw.Bytes,
	})
	if err != nil {
		return otherRecipientInfo{}, err
	}
	var other otherRecipientInfo
	rest, err := asn1.Unmarshal(sequenceDER, &other)
	if err != nil {
		return otherRecipientInfo{}, fmt.Errorf("pkcs7: malformed OtherRecipientInfo: %w", err)
	}
	if len(rest) != 0 {
		return otherRecipientInfo{}, errors.New("pkcs7: trailing data in OtherRecipientInfo")
	}
	return other, nil
}

func deriveKEMKey(sharedSecret []byte, wrap pkix.AlgorithmIdentifier, kekLength int, ukm []byte) ([]byte, error) {
	infoDER, err := asn1.Marshal(cmsORIForKEMOtherInfo{
		Wrap:      wrap,
		KEKLength: kekLength,
		UKM:       ukm,
	})
	if err != nil {
		return nil, err
	}
	return hkdf.Key(sha256.New, sharedSecret, nil, string(infoDER), kekLength)
}

func decryptKEMRecipientKey(recipients []asn1.RawValue, cert *x509.Certificate, privateKey any) ([]byte, error) {
	expectedKEM, decapsulate, ok := mlKEMDecapsulator(privateKey)
	if !ok {
		return nil, ErrUnsupportedAlgorithm
	}

	for _, raw := range recipients {
		if raw.Class != 2 || raw.Tag != 4 {
			continue
		}
		other, err := parseOtherRecipientInfo(raw)
		if err != nil {
			return nil, err
		}
		if !other.Type.Equal(OIDOtherRecipientInfoKEM) {
			continue
		}

		var info kemRecipientInfo
		rest, err := asn1.Unmarshal(other.Value.FullBytes, &info)
		if err != nil {
			return nil, fmt.Errorf("pkcs7: malformed KEMRecipientInfo: %w", err)
		}
		if len(rest) != 0 {
			return nil, errors.New("pkcs7: trailing data in KEMRecipientInfo")
		}
		matches, err := recipientIdentifierMatchesCertificate(info.RecipientID, cert)
		if err != nil {
			return nil, err
		}
		if !matches {
			continue
		}
		if info.Version != 0 {
			return nil, errors.New("pkcs7: KEMRecipientInfo version must be 0")
		}
		if !info.KEM.Algorithm.Equal(expectedKEM) {
			return nil, errors.New("pkcs7: KEMRecipientInfo algorithm does not match private key")
		}
		if algorithmIdentifierHasParameters(info.KEM) {
			return nil, errors.New("pkcs7: ML-KEM AlgorithmIdentifier parameters must be absent")
		}
		if !info.KDF.Algorithm.Equal(OIDKDFHKDFSHA256) || algorithmIdentifierHasParameters(info.KDF) {
			return nil, errors.New("pkcs7: unsupported or malformed KEMRecipientInfo KDF")
		}
		if !info.Wrap.Algorithm.Equal(OIDKeyWrapAES256) || algorithmIdentifierHasParameters(info.Wrap) || info.KEKLength != mlKEMKEKLength {
			return nil, errors.New("pkcs7: unsupported or malformed KEMRecipientInfo key wrap")
		}

		sharedSecret, err := decapsulate(info.Ciphertext)
		if err != nil {
			return nil, fmt.Errorf("pkcs7: ML-KEM decapsulation failed: %w", err)
		}
		kek, err := deriveKEMKey(sharedSecret, info.Wrap, info.KEKLength, info.UKM)
		clear(sharedSecret)
		if err != nil {
			return nil, err
		}
		contentKey, err := aesKeyUnwrap(kek, info.EncryptedKey)
		clear(kek)
		if err != nil {
			return nil, err
		}
		return contentKey, nil
	}
	return nil, errors.New("pkcs7: no enveloped recipient for provided certificate")
}

func mlKEMDecapsulator(privateKey any) (asn1.ObjectIdentifier, func([]byte) ([]byte, error), bool) {
	switch privateKey := privateKey.(type) {
	case *mlkem.DecapsulationKey768:
		return OIDKeyAlgorithmMLKEM768, privateKey.Decapsulate, true
	case *mlkem.DecapsulationKey1024:
		return OIDKeyAlgorithmMLKEM1024, privateKey.Decapsulate, true
	default:
		return nil, nil, false
	}
}

func recipientIdentifierMatchesCertificate(identifier asn1.RawValue, cert *x509.Certificate) (bool, error) {
	if cert == nil {
		return false, errors.New("pkcs7: recipient certificate is nil")
	}
	switch {
	case identifier.Class == 0 && identifier.Tag == asn1.TagSequence:
		var issuerAndSerial issuerAndSerial
		rest, err := asn1.Unmarshal(identifier.FullBytes, &issuerAndSerial)
		if err != nil {
			return false, fmt.Errorf("pkcs7: malformed issuerAndSerialNumber: %w", err)
		}
		if len(rest) != 0 {
			return false, errors.New("pkcs7: trailing data in issuerAndSerialNumber")
		}
		return isCertMatchForIssuerAndSerial(cert, issuerAndSerial), nil
	case identifier.Class == 2 && identifier.Tag == 0:
		if len(identifier.Bytes) == 0 || len(cert.SubjectKeyId) == 0 {
			return false, nil
		}
		return subtle.ConstantTimeCompare(identifier.Bytes, cert.SubjectKeyId) == 1, nil
	default:
		return false, errors.New("pkcs7: unsupported RecipientIdentifier")
	}
}

func algorithmIdentifierHasParameters(identifier pkix.AlgorithmIdentifier) bool {
	return len(identifier.Parameters.FullBytes) != 0
}

func aesKeyWrap(kek, plaintext []byte) ([]byte, error) {
	if len(plaintext) < 16 || len(plaintext)%8 != 0 {
		return nil, errors.New("pkcs7: AES key wrap input must be at least 16 bytes and a multiple of 8 bytes")
	}
	block, err := aes.NewCipher(kek)
	if err != nil {
		return nil, err
	}

	n := len(plaintext) / 8
	output := make([]byte, len(plaintext)+8)
	copy(output[:8], aesKeyWrapInitialValue[:])
	copy(output[8:], plaintext)
	buffer := make([]byte, aes.BlockSize)
	for j := 0; j < 6; j++ {
		for i := 1; i <= n; i++ {
			copy(buffer[:8], output[:8])
			copy(buffer[8:], output[i*8:(i+1)*8])
			block.Encrypt(buffer, buffer)
			t := uint64(n*j + i)
			binary.BigEndian.PutUint64(output[:8], binary.BigEndian.Uint64(buffer[:8])^t)
			copy(output[i*8:(i+1)*8], buffer[8:])
		}
	}
	clear(buffer)
	return output, nil
}

func aesKeyUnwrap(kek, ciphertext []byte) ([]byte, error) {
	if len(ciphertext) < 24 || len(ciphertext)%8 != 0 {
		return nil, errors.New("pkcs7: malformed AES key wrap ciphertext")
	}
	block, err := aes.NewCipher(kek)
	if err != nil {
		return nil, err
	}

	n := len(ciphertext)/8 - 1
	working := append([]byte(nil), ciphertext...)
	buffer := make([]byte, aes.BlockSize)
	for j := 5; j >= 0; j-- {
		for i := n; i >= 1; i-- {
			t := uint64(n*j + i)
			binary.BigEndian.PutUint64(buffer[:8], binary.BigEndian.Uint64(working[:8])^t)
			copy(buffer[8:], working[i*8:(i+1)*8])
			block.Decrypt(buffer, buffer)
			copy(working[:8], buffer[:8])
			copy(working[i*8:(i+1)*8], buffer[8:])
		}
	}
	clear(buffer)
	if subtle.ConstantTimeCompare(working[:8], aesKeyWrapInitialValue[:]) != 1 {
		clear(working)
		return nil, errors.New("pkcs7: AES key unwrap integrity check failed")
	}
	plaintext := append([]byte(nil), working[8:]...)
	clear(working)
	return plaintext, nil
}
