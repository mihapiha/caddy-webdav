package webdav

import (
	"encoding/xml"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/pkg/xattr"
	webdav "golang.org/x/net/webdav"
)

var testFile *os.File

func TestMain(m *testing.M) {
	f, err := os.CreateTemp(".", "example-test.txt")
	testFile = f
	if err != nil {
		fmt.Println("Could not open test file")
		os.Exit(1)
	}
		
	retCode := m.Run()

	os.Remove(testFile.Name()) // clean up
	os.Exit(retCode)
}

func Test_DeadProps(t *testing.T) {
	for _, tt := range []struct {
		explantion string
		tags []string
		values []string
		propsExpected map[xml.Name]webdav.Property
		errExpected error
	}{
		{
			"regular",
			[]string{xattrPrefix+"example.com/ns:attr"},
			[]string{"foobar"},
			map[xml.Name]webdav.Property{
				xml.Name{Space: "example.com/ns", Local: "attr"}: webdav.Property{XMLName: xml.Name{Space: "example.com/ns", Local: "attr"}, InnerXML: []byte("foobar")}},
			nil,
		},
	}{
		t.Run(tt.explantion, func(t *testing.T) {
			for i := range tt.tags {
				tag := tt.tags[i]
				val := []byte(tt.values[i])

				err := xattr.FSet(testFile, tag, val)
				if err != nil {
					t.Fatalf("Error setting tag in test. Supported: %v, err: %v", xattr.XATTR_SUPPORTED, err)
				}
			}

			xFile := FileXattr{testFile}
			
			props, err := xFile.DeadProps()

			if got, want := err, tt.errExpected; got != want {
				t.Fatalf("err=%v, want=%v", got, want)
			}

			if got, want := props, tt.propsExpected; !reflect.DeepEqual(got, want) {
				t.Fatalf("props=%v, want=%v", got, want)
			}
			
			for _, tag := range tt.tags {
				xattr.FRemove(testFile, tag)
			}
		})
	}
}

func Test_parsePropName(t *testing.T) {
	for _, tt := range []struct {
		explanation      string
		input            string
		propNameExpected xml.Name
		okExpected       bool
	}{
		{
			"regular namespaced name",
			xattrPrefix+"example.com/ns:attr",
			xml.Name{Space: "example.com/ns", Local: "attr"},
			true,
		},
		{
			"regular namespaced name, more colons",
			xattrPrefix+"https://example.com/ns:attr",
			xml.Name{Space: "https://example.com/ns", Local: "attr"},
			true,
		},
		{
			"regular namespaced name, even more colons",
			xattrPrefix+"https://example.com:9000/ns:attr",
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
			xattrPrefix+"example.com/ns_attr",
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
