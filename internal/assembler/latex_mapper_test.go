package assembler

import (
	"strings"
	"testing"

	"github.com/tidwall/gjson"
)

func TestLaTeXMapper_InjectMacros(t *testing.T) {
	jsonPayload := []byte(`{"brand": "Zippee", "awb": "123", "meta": {"weight": "500g", "special": "%&$#"}}`)

	expectedMacros := []string{
		`\def\brand{Zippee}`,
		`\def\awb{123}`,
		`\def\metaWeight{500g}`,
		`\def\metaSpecial{\%\&\$\#}`,
	}

	macros, err := MapToMacros(jsonPayload)
	if err != nil {
		t.Fatalf("MapToMacros failed: %v", err)
	}

	for _, expected := range expectedMacros {
		if !strings.Contains(macros, expected) {
			t.Errorf("Expected macros to contain %s, got:\n%s", expected, macros)
		}
	}

	parsed := gjson.ParseBytes(jsonPayload)
	macrosParsed, err := MapParsedToMacros(parsed)
	if err != nil {
		t.Fatalf("MapParsedToMacros failed: %v", err)
	}

	if macros != macrosParsed {
		t.Fatalf("MapParsedToMacros returned different result than MapToMacros:\n%s\nvs\n%s", macrosParsed, macros)
	}
}
