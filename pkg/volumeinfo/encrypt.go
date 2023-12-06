package volumeinfo

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
)

func getEncryptionKey(key []byte) []byte {
	aesKey := make([]byte, 32)

	// length must be 32 bytes for AES-256
	copy(aesKey, key)
	return aesKey
}

func encrypt(plaintext []byte, key []byte) ([]byte, error) {
	aesKey := getEncryptionKey(key)
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}

	//Create a new GCM
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	//Create a nonce. Nonce should be from GCM
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	//Encrypt the data using aesGCM.Seal
	//Since we don't want to save the nonce somewhere else in this case, we add it as a prefix to the encrypted data. The first nonce argument in Seal is the prefix.
	ciphertext := aesGCM.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

func decrypt(encryptedtext []byte, key []byte) ([]byte, error) {
	//Create a new Cipher Block from the key
	aesKey := getEncryptionKey(key)
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}

	//Create a new GCM
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	//Get the nonce size
	nonceSize := aesGCM.NonceSize()

	//Extract the nonce from the encrypted data
	nonce, ciphertext := encryptedtext[:nonceSize], encryptedtext[nonceSize:]

	//Decrypt the data
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}
