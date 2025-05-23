package main

import "strings"

type Translator struct {
	mac   map[string]string
	radio map[string]string
}

func (t *Translator) TranslateMac(mac string) string {
	for key, value := range t.mac {
		if strings.EqualFold(mac, key) {
			return value
		}
	}

	return strings.ToUpper(mac)
}

func (t *Translator) TranslateRadio(radio string) string {
	for key, value := range t.radio {
		if strings.EqualFold(radio, key) {
			return value
		}
	}

	return radio
}
