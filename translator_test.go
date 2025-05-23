package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTranslator_TranslateMac(t *testing.T) {
	trans := Translator{
		mac: map[string]string{
			"AA:AA:AA:AA:AA:01": "iphone",
			"aa:aa:aa:aa:aa:02": "ipad",
		},
	}

	assert.Equal(t, "iphone", trans.TranslateMac("AA:AA:AA:AA:AA:01"))
	assert.Equal(t, "iphone", trans.TranslateMac("aa:aa:aa:aa:aa:01")) // case-insensitive
	assert.Equal(t, "ipad", trans.TranslateMac("aa:aa:aa:aa:aa:02"))
	assert.Equal(t, "BB:BB:BB:BB:BB:01", trans.TranslateMac("BB:BB:BB:BB:BB:01"))
	assert.Equal(t, "BB:BB:BB:BB:BB:08", trans.TranslateMac("bb:bb:bb:bb:bb:08")) // capitalization
}

func TestTranslator_TranslateRadio(t *testing.T) {
	trans := Translator{
		radio: map[string]string{
			"radio0": "wifi_2.4",
			"radio1": "wifi_5",
		},
	}

	assert.Equal(t, "wifi_2.4", trans.TranslateRadio("radio0"))
	assert.Equal(t, "wifi_5", trans.TranslateRadio("radio1"))
	assert.Equal(t, "unknown_radio", trans.TranslateRadio("unknown_radio"))
}
