package keybox

// Keybox
// Tests
// Copyright © 2018-2024 Eduard Sesigin. All rights reserved. Contacts: <claygod@yandex.ru>

import (
	"testing"
)

var forTestPass = "12345"

func TestNewKeybox(t *testing.T) {
	kb, err := New([]byte(forTestPass))
	if err != nil {
		t.Error(err)
	}

	if string(kb.pass) == forTestPass {
		t.Errorf("the password was not encrypted")
	}

	if pass2 := kb.Key(); forTestPass != string(pass2) {
		t.Errorf("want `%s` have `%s`", forTestPass, string(pass2))
	}
}

func TestNewWithClean(t *testing.T) {
	pass := []byte(forTestPass)

	kb, err := NewWithClean(pass)
	if err != nil {
		t.Error(err)
	}

	if string(pass) == forTestPass {
		t.Errorf("after creation the original must be erased")
	}

	if pass2 := kb.Key(); forTestPass != string(pass2) {
		t.Errorf("want `%s` have `%s`", forTestPass, string(pass2))
	}
}

func TestNewWithCleanMax(t *testing.T) {
	pass := make([]byte, lenPassMax*2)

	_, err := NewWithClean(pass)
	if err == nil {
		t.Error("want error")
	}
}

func TestNewWithCleanMin(t *testing.T) {
	pass := make([]byte, lenPassMin-1)

	_, err := NewWithClean(pass)
	if err == nil {
		t.Error("want error")
	}
}

func TestEncrypt(t *testing.T) {
	kb := KeyBox{}
	key := make([]byte, lenKey)
	for i := 0; i < lenKey; i++ {
		key[i] = 1
	}

	res, err := kb.encrypt([]byte(forTestPass), key)
	if err != nil {
		t.Error(err)
	}

	kb.rndKey = key
	kb.pass = res

	res2, err := kb.decrypt()
	if err != nil {
		t.Error(err)
	}

	if string(res2) != forTestPass {
		t.Errorf("want `%s` have `%s`", forTestPass, string(res2))
	}
}

func TestKeyboxMax(t *testing.T) {
	pass1 := make([]byte, lenPassMax*2)

	_, err := New([]byte(pass1))
	if err == nil {
		t.Error("an error is expected because the length exceeds the allowed limit")
	}
}

func TestKeyboxMin(t *testing.T) {
	pass1 := make([]byte, lenPassMin-1)

	_, err := New([]byte(pass1))
	if err == nil {
		t.Error("an error is expected because the length is less than the allowed length")
	}
}
