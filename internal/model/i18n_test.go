package model

import "testing"

func TestReasonLabel_AllCodes(t *testing.T) {
	for _, code := range AllReasonCodes() {
		if ReasonLabel(code, "ru") == "" || ReasonLabel(code, "ru") == string(code) {
			t.Errorf("missing RU label for %s", code)
		}
		if ReasonLabel(code, "en") == "" || ReasonLabel(code, "en") == string(code) {
			t.Errorf("missing EN label for %s", code)
		}
	}
}
