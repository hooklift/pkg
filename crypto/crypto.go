package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"io"

	"github.com/golang/glog"
	"golang.org/x/crypto/scrypt"
)

// HashPassword returns a Scrypt key of a password, encoded in hexadecimal.
func HashPassword(password, salt string) string {
	if salt == "" {
		glog.Fatal("password salt can't be empty")
	}
	// salt is a fixed random string. The primary function of salts is to defend
	// against dictionary attacks versus a list of password hashes and against
	// pre-computed rainbow table attacks.
	saltBytes := []byte(salt)

	// Scrypt parameters are automatically stored in the resulting key, so
	// they don't need to be stored separately.
	// We have to keep in mind that if we ever need to change parameters, we need
	// to keep backwards compatibility with existing passwords. So, we will have
	// to keep around previous hashing functions too and force users to change their
	// passwords.
	//
	// The recommended parameters for interactive logins as of 2009 are N=16384,
	// r=8, p=1. They should be increased as memory latency and CPU parallelism
	// increases. Remember to get a good random salt. More information about how to better
	// tweak scrypt work factors can be found at http://stackoverflow.com/questions/11126315/what-are-optimal-scrypt-work-factors
	//
	// For more general information about Scrypt refer to http://en.wikipedia.org/wiki/Scrypt/
	k, err := scrypt.Key([]byte(password), saltBytes, 16384, 8, 1, 32)
	if err != nil {
		glog.Errorf("Wrong scrypt parameters: %v", err)
	}

	return hex.EncodeToString(k)
}

// RandBytes generates a random byte slice of the given size.
func RandBytes(size int) []byte {
	p := make([]byte, size)
	_, err := io.ReadFull(rand.Reader, p)
	if err != nil {
		p = nil
	}

	return p
}

// KeySize is size of AES-256-GCM keys in bytes.
const KeySize = 32

// const nonceSize = 24

// NewKey randomly generates a new key.
func NewKey() []byte {
	return RandBytes(KeySize)
}

// Encrypt encrypts the message using AES-GCM with the given key.
// TODO(c4milo): Rotate encryption key
// TODO(c4milo): Add key ID to additional data parameter when sealing.
func Encrypt(key, message []byte) ([]byte, bool) {
	c, err := aes.NewCipher(key)
	if err != nil {
		glog.Errorf("failed creating new cipher: %#v", err)
		return nil, false
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		glog.Errorf("failed creating new GCM: %#v", err)
		return nil, false
	}
	iv := RandBytes(gcm.NonceSize())
	if iv == nil {
		glog.Error("failed reading random value")
		return nil, false
	}

	// We wrap the ciphertext with the initialization vector (IV) to minimize
	// guessing attacks a bit more.
	gcm.Seal(message, iv, message, nil)
	iv = append(iv, message...)

	output := make([]byte, hex.EncodedLen(len(iv)))
	hex.Encode(output, iv)
	return output, true
}

// Decrypt decrypts the message and removes any padding.
func Decrypt(key, message []byte) ([]byte, bool) {
	output := make([]byte, hex.DecodedLen(len(message)))
	_, err := hex.Decode(output, message)
	if err != nil {
		return nil, false
	}

	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, false
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, false
	}

	nonceSize := gcm.NonceSize()
	if len(output) < nonceSize {
		return nil, false
	}
	gcm.Open(output[nonceSize:], output[:nonceSize], output[nonceSize:], nil)
	return output[nonceSize:], true
}
