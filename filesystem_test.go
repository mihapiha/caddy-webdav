package webdav

import (
	"encoding/xml"
	"fmt"
	"os"
	"reflect"
	"slices"
	"strings"
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
		explanation   string
		tags          []string
		values        []string
		propsExpected map[xml.Name]webdav.Property
		errExpected   error
	}{
		{
			"regular",
			[]string{xattrPrefix + "example.com/ns:attr"},
			[]string{"foobar"},
			map[xml.Name]webdav.Property{
				xml.Name{Space: "example.com/ns", Local: "attr"}: webdav.Property{XMLName: xml.Name{Space: "example.com/ns", Local: "attr"}, InnerXML: []byte("foobar")}},
			nil,
		},
		{
			"regular two tags",
			[]string{xattrPrefix + "example.com/ns:attr", xattrPrefix + "example.com/ns:name"},
			[]string{"foobar", "foobaz"},
			map[xml.Name]webdav.Property{
				xml.Name{Space: "example.com/ns", Local: "attr"}: webdav.Property{XMLName: xml.Name{Space: "example.com/ns", Local: "attr"}, InnerXML: []byte("foobar")},
				xml.Name{Space: "example.com/ns", Local: "name"}: webdav.Property{XMLName: xml.Name{Space: "example.com/ns", Local: "name"}, InnerXML: []byte("foobaz")},
			},
			nil,
		},
		{
			"first missing attr name, second ok",
			[]string{xattrPrefix + "example.com/ns:", xattrPrefix + "example.com/ns:name"},
			[]string{"foobar", "foobaz"},
			map[xml.Name]webdav.Property{
				xml.Name{Space: "example.com/ns", Local: "name"}: webdav.Property{XMLName: xml.Name{Space: "example.com/ns", Local: "name"}, InnerXML: []byte("foobaz")},
			},
			nil,
		},
		{
			"first missing attr name no colon, second ok",
			[]string{xattrPrefix + "example.com/ns", xattrPrefix + "example.com/ns:name"},
			[]string{"foobar", "foobaz"},
			map[xml.Name]webdav.Property{
				xml.Name{Space: "example.com/ns", Local: "name"}: webdav.Property{XMLName: xml.Name{Space: "example.com/ns", Local: "name"}, InnerXML: []byte("foobaz")},
			},
			nil,
		},
		{
			"first ok, second missing attr name",
			[]string{xattrPrefix + "example.com/ns:name", xattrPrefix + "example.com/ns:"},
			[]string{"foobaz", "foobar"},
			map[xml.Name]webdav.Property{
				xml.Name{Space: "example.com/ns", Local: "name"}: webdav.Property{XMLName: xml.Name{Space: "example.com/ns", Local: "name"}, InnerXML: []byte("foobaz")},
			},
			nil,
		},
		{
			"first ok, second missing attr name no colon",
			[]string{xattrPrefix + "example.com/ns:name", xattrPrefix + "example.com/ns"},
			[]string{"foobaz", "foobar"},
			map[xml.Name]webdav.Property{
				xml.Name{Space: "example.com/ns", Local: "name"}: webdav.Property{XMLName: xml.Name{Space: "example.com/ns", Local: "name"}, InnerXML: []byte("foobaz")},
			},
			nil,
		},
		{
			"first ok, second missing prefix",
			[]string{xattrPrefix + "example.com/ns:name", "example.com/ns:attr"},
			[]string{"foobaz", "foobar"},
			map[xml.Name]webdav.Property{
				xml.Name{Space: "example.com/ns", Local: "name"}: webdav.Property{XMLName: xml.Name{Space: "example.com/ns", Local: "name"}, InnerXML: []byte("foobaz")},
			},
			nil,
		},
		{
			"first missing prefix, second",
			[]string{"example.com/ns:name", xattrPrefix + "example.com/ns:attr"},
			[]string{"foobaz", "foobar"},
			map[xml.Name]webdav.Property{
				xml.Name{Space: "example.com/ns", Local: "attr"}: webdav.Property{XMLName: xml.Name{Space: "example.com/ns", Local: "attr"}, InnerXML: []byte("foobar")},
			},
			nil,
		},
		{
			"both missing prefix",
			[]string{"example.com/ns:name", "example.com/ns:attr"},
			[]string{"foobaz", "foobar"},
			map[xml.Name]webdav.Property{},
			nil,
		},
	} {
		t.Run(tt.explanation, func(t *testing.T) {
			for i := range tt.tags {
				tag := tt.tags[i]

				if !strings.HasPrefix(tag, xattrPrefix) {
					continue
				}

				val := []byte(tt.values[i])

				err := xattr.FSet(testFile, tag, val)
				if err != nil {
					t.Fatalf("Error setting tag in test. Supported: %v, err: %v", xattr.XATTR_SUPPORTED, err)
				}
			}

			defer func() {
				for _, tag := range tt.tags {
					if !strings.HasPrefix(tag, xattrPrefix) {
						continue
					}

					xattr.FRemove(testFile, tag)
				}
			}()

			stat, err := testFile.Stat()
			if err != nil {
				t.Fatalf("err getting file stat: %v", err)
			}

			xFile := FileXattr{testFile, "./" + stat.Name()}

			props, err := xFile.DeadProps()

			if got, want := err, tt.errExpected; got != want {
				t.Fatalf("err=%v, want=%v", got, want)
			}

			if got, want := props, tt.propsExpected; !reflect.DeepEqual(got, want) {
				t.Fatalf("props=%v, want=%v", got, want)
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
			xattrPrefix + "example.com/ns:attr",
			xml.Name{Space: "example.com/ns", Local: "attr"},
			true,
		},
		{
			"regular namespaced name, more colons",
			xattrPrefix + "https://example.com/ns:attr",
			xml.Name{Space: "https://example.com/ns", Local: "attr"},
			true,
		},
		{
			"regular namespaced name, even more colons",
			xattrPrefix + "https://example.com:9000/ns:attr",
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
			xattrPrefix + "example.com/ns_attr",
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

func Test_DeadPropsPatch(t *testing.T) {
	for _, tt := range []struct {
		explanation   string
		patches       []webdav.Proppatch
		output        []webdav.Propstat
		attrsExpected []string
		errExpected   error
	}{
		{
			"set 1 attr",
			[]webdav.Proppatch{
				{
					Remove: false,
					Props:  []webdav.Property{{XMLName: xml.Name{Space: "example.com", Local: "tag"}}},
				},
			},
			[]webdav.Propstat{
				{
					Status: 201,
					Props:  []webdav.Property{{XMLName: xml.Name{Space: "example.com", Local: "tag"}}},
				},
			},
			[]string{xattrPrefix + "example.com:tag"},
			nil,
		},
		{
			"set 1, remove 1 attr",
			[]webdav.Proppatch{
				{
					Remove: false,
					Props:  []webdav.Property{{XMLName: xml.Name{Space: "example.com", Local: "tag"}}},
				},
				{
					Remove: true,
					Props:  []webdav.Property{{XMLName: xml.Name{Space: "example.com", Local: "tag"}}},
				},
			},
			[]webdav.Propstat{
				{
					Status: 201,
					Props:  []webdav.Property{{XMLName: xml.Name{Space: "example.com", Local: "tag"}}},
				},
				{
					Status: 200,
					Props:  []webdav.Property{{XMLName: xml.Name{Space: "example.com", Local: "tag"}}},
				},
			},
			[]string{},
			nil,
		},
		{
			"remove nonexisting",
			[]webdav.Proppatch{
				{
					Remove: true,
					Props:  []webdav.Property{{XMLName: xml.Name{Space: "example.com", Local: "tag"}}},
				},
			},
			[]webdav.Propstat{
				{
					Status: 500,
					Props:  []webdav.Property{{XMLName: xml.Name{Space: "example.com", Local: "tag"}}},
				},
			},
			[]string{},
			nil,
		},
		{
			"set 2, remove 1 attr",
			[]webdav.Proppatch{
				{
					Remove: false,
					Props:  []webdav.Property{{XMLName: xml.Name{Space: "example.com", Local: "tag"}}},
				},
				{
					Remove: true,
					Props:  []webdav.Property{{XMLName: xml.Name{Space: "example.com", Local: "tag"}}},
				},
				{
					Remove: false,
					Props:  []webdav.Property{{XMLName: xml.Name{Space: "https://example.org", Local: "tag"}}},
				},
			},
			[]webdav.Propstat{
				{
					Status: 201,
					Props:  []webdav.Property{{XMLName: xml.Name{Space: "example.com", Local: "tag"}}},
				},
				{
					Status: 200,
					Props:  []webdav.Property{{XMLName: xml.Name{Space: "example.com", Local: "tag"}}},
				},
				{
					Status: 201,
					Props:  []webdav.Property{{XMLName: xml.Name{Space: "https://example.org", Local: "tag"}}},
				},
			},
			[]string{xattrPrefix + "https://example.org:tag"},
			nil,
		},
	} {
		t.Run(tt.explanation, func(t *testing.T) {
			stat, err := testFile.Stat()
			if err != nil {
				t.Fatalf("err getting file stat: %v", err)
			}

			xFile := FileXattr{testFile, "./" + stat.Name()}
			propstat, err := xFile.Patch(tt.patches)

			xattrs, err := xattr.FList(testFile)
			if err != nil {
				t.Fatalf("err getting xattrs: %v", err)
			}

			defer func() {
				for _, attr := range xattrs {
					xattr.FRemove(testFile, attr)
				}
			}()

			if got, want := err, tt.errExpected; got != want {
				t.Fatalf("err=%v, want=%v", got, want)
			}

			for i := range propstat {
				//ignore err description
				propstat[i].ResponseDescription = ""
			}

			if got, want := propstat, tt.output; !reflect.DeepEqual(got, want) {
				t.Fatalf("propstat=%v, want=%v", got, want)
			}

			if got, want := xattrs, tt.attrsExpected; !slices.Equal(got, want) {
				t.Fatalf("xattrs=%v, want=%v", got, want)
			}

		})
	}
}
