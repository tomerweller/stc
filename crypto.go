
package main

import (
	"fmt"
	"os"
	"crypto"
	"crypto/rand"
	"crypto/sha256"
	"golang.org/x/crypto/ed25519"
)

func XdrSHA256(t XdrAggregate) []byte {
	sha := sha256.New()
	t.XdrMarshal(&XdrOut{sha}, "")
	return sha.Sum(nil)
}

func TxPayloadHash(network string, e *TransactionEnvelope) []byte {
	payload := TransactionSignaturePayload{
		NetworkId: sha256.Sum256(([]byte)(network)),
	}
	payload.TaggedTransaction.Type = ENVELOPE_TYPE_TX
	*payload.TaggedTransaction.Tx() = e.Tx
	return XdrSHA256(&payload)
}

func (pk *PublicKey) Verify(message, sig []byte) bool {
	switch pk.Type {
	case PUBLIC_KEY_TYPE_ED25519:
		return ed25519.Verify(pk.Ed25519()[:], message, sig)
	default:
		return false
	}
}

type Ed25519Priv ed25519.PrivateKey

func (sk Ed25519Priv) String() string {
	return ToStrKey(STRKEY_SEED_ED25519, ed25519.PrivateKey(sk).Seed())
}

func (sk Ed25519Priv) Sign(msg []byte) ([]byte, error) {
	return ed25519.PrivateKey(sk).Sign(rand.Reader, msg, crypto.Hash(0))
}

func (sk Ed25519Priv) Public() *PublicKey {
	ret := PublicKey{ Type: PUBLIC_KEY_TYPE_ED25519 }
	copy(ret.Ed25519()[:], ed25519.PrivateKey(sk).Public().(ed25519.PublicKey))
	return &ret
}

// Use struct instead of interface so we can have Scan method
type PrivateKey struct {
	k interface {
		String() string
		Sign([]byte) ([]byte, error)
		Public() *PublicKey
	}
}
func (sk PrivateKey) String() string { return sk.k.String() }
func (sk PrivateKey) Sign(msg []byte) ([]byte, error) { return sk.k.Sign(msg) }
func (sk PrivateKey) Public() *PublicKey { return sk.k.Public() }

func (sec *PrivateKey) Scan(ss fmt.ScanState, _ rune) error {
	bs, err := ss.Token(true, isKeyChar)
	if err != nil {
		return err
	}
	key, vers := FromStrKey(string(bs))
	switch vers {
	case STRKEY_SEED_ED25519:
		sec.k = Ed25519Priv(ed25519.NewKeyFromSeed(key))
		return nil
	default:
		return StrKeyError("Invalid private key")
	}
}

func (sec *PrivateKey) SignTx(network string, e *TransactionEnvelope) error {
	sig, err := sec.Sign(TxPayloadHash(network, e))
	if err != nil {
		return err
	}

	e.Signatures = append(e.Signatures, DecoratedSignature{
		Hint: sec.Public().Hint(),
		Signature: sig,
	})
	return nil
}

func genEd25519() PrivateKey {
	_, sk, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	return PrivateKey{ Ed25519Priv(sk) }
}

func KeyGen(pkt PublicKeyType) PrivateKey {
	switch pkt {
	case PUBLIC_KEY_TYPE_ED25519:
		return genEd25519()
	default:
		panic(fmt.Sprintf("KeyGen: unsupported PublicKeyType %v", pkt))
	}
}
