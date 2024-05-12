package keybox

// Keybox
// Main
// Copyright © 2018-2024 Eduard Sesigin. All rights reserved. Contacts: <claygod@yandex.ru>

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
)

/*
KeyBox - to store the password.
*/
type KeyBox struct {
	rndKey []byte // for safety, keep separate
	pass   []byte
}

func New(pass []byte) (*KeyBox, error) {
	key := make([]byte, 16)
	rand.Read(key)

	k := &KeyBox{}
	k.rndKey = key
	cpass, err := k.encrypt(pass, key)
	if err != nil {
		return nil, err
	}
	k.pass = cpass
	if _, err := k.decrypt(); err != nil { // reverse check
		return nil, err
	}
	return k, nil
}

func (k *KeyBox) Key() []byte {
	pass, _ := k.decrypt()
	return pass
}

func (k *KeyBox) encrypt(ptxt []byte, key []byte) ([]byte, error) {
	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, ptxt, nil), nil
}

func (k *KeyBox) decrypt() ([]byte, error) {
	c, err := aes.NewCipher(k.rndKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(k.pass) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := k.pass[:nonceSize], k.pass[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}
