package keybox

// Keybox
// Tests
// Copyright © 2023 Eduard Sesigin. All rights reserved. Contacts: <claygod@yandex.ru>

import (
	"fmt"
	"testing"
)

/*
TestNewKeybox - easy keybox test
*/
func TestNewKeybox(t *testing.T) {
	pass1 := "12345"

	kb, err := New([]byte(pass1))

	if err != nil {
		t.Error(err)
	}

	if pass2 := kb.Key(); pass1 != string(pass2) {
		t.Errorf("want `%s` have `%s`", pass1, string(pass2))
	}

	fmt.Printf("in memory: `%s`\n", kb.pass)
}
