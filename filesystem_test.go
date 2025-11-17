package webdav

import (
	"encoding/xml"
	"fmt"
	"testing"
)

func Test_parsePropName(t *testing.T) {
	for _, tt := range []struct {
		explanation      string
		input            string
		propNameExpected xml.Name
		okExpected       bool
	}{
		{
			"regular namespaced name",
			"dav:example.com/ns:attr",
			xml.Name{Space: "example.com/ns", Local: "attr"},
			true,
		},
		{
			"regular namespaced name, more colons",
			"dav:https://example.com/ns:attr",
			xml.Name{Space: "https://example.com/ns", Local: "attr"},
			true,
		},
		{
			"regular namespaced name, even more colons",
			"dav:https://example.com:9000/ns:attr",
			xml.Name{Space: "https://example.com:9000/ns", Local: "attr"},
			true,
		},
		{
			"missing dav prefix",
			"https://example.com/ns:attr",
			xml.Name{},
			false,
		},
		{
			"missing suffix",
			"dav:example.com/ns_attr",
			xml.Name{},
			false,
		},
	} {
		t.Run(fmt.Sprintf("%s [%s]", tt.explanation, tt.input), func(t *testing.T) {
			propName, ok := parsePropName(tt.input)

			if got, want := ok, tt.okExpected; got != want {
				t.Fatalf("ok=%v, want=%v", got, want)
			}

			if got, want := propName, tt.propNameExpected; got != want {
				t.Fatalf("propName=%v, want=%v", got, want)
			}

		})
	}
}
