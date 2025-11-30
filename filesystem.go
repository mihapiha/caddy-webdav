package webdav

import (
	"context"
	"encoding/xml"
	"fmt"
	"strings"

	"os"

	xattr "github.com/pkg/xattr"
	webdav "golang.org/x/net/webdav"
)

var _ webdav.FileSystem = (*WrapFS)(nil)

type WrapFS struct {
	fileSystem webdav.Dir
}

func (fs WrapFS) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
	return fs.fileSystem.Mkdir(ctx, name, perm)
}
func (fs WrapFS) OpenFile(ctx context.Context, name string, flag int, perm os.FileMode) (webdav.File, error) {
	file, err := fs.fileSystem.OpenFile(ctx, name, flag, perm)
	wrapped := FileXattr{file}
	return wrapped, err
}
func (fs WrapFS) RemoveAll(ctx context.Context, name string) error {
	return fs.fileSystem.RemoveAll(ctx, name)
}
func (fs WrapFS) Rename(ctx context.Context, oldName, newName string) error {
	return fs.fileSystem.Rename(ctx, oldName, newName)
}
func (fs WrapFS) Stat(ctx context.Context, name string) (os.FileInfo, error) {
	return fs.fileSystem.Stat(ctx, name)
}

var _ webdav.File = (*FileXattr)(nil)
var _ webdav.DeadPropsHolder = (*FileXattr)(nil)

type FileXattr struct {
	webdav.File
}

const xattrPrefix = "user.dav:"

func parsePropName(xattrName string) (xml.Name, bool) {
	propName, ok := strings.CutPrefix(xattrName, xattrPrefix)

	if !ok {
		return xml.Name{}, false
	}

	parts := strings.Split(propName, ":")

	if len(parts) < 2 {
		return xml.Name{}, false
	}

	namespace := strings.Join(parts[0:len(parts)-1], ":")
	localName := parts[len(parts)-1]

	if namespace == "" || localName == "" {
		return xml.Name{}, false
	}

	return xml.Name{Space: namespace, Local: localName}, true
}

func (f FileXattr) DeadProps() (map[xml.Name]webdav.Property, error) {
	props := make(map[xml.Name]webdav.Property)
	fstat, err := f.File.Stat()
	if err != nil {
		return nil, err
	}

	xattrNames, err := xattr.List(fstat.Name())
	if err != nil {
		return nil, err
	}

	for _, xattrName := range xattrNames {

		propName, ok := parsePropName(xattrName)
		if !ok {
			//this xattr is not for webdav use
			continue
		}

		attr, err := xattr.Get(fstat.Name(), xattrName)
		if err != nil {
			return nil, err
		}

		props[propName] = webdav.Property{XMLName: propName, InnerXML: attr}
	}

	return props, nil
}

func propertyToAttr(prop webdav.Property) string {
	return fmt.Sprintf("%v%v:%v", xattrPrefix, prop.XMLName.Space, prop.XMLName.Local)
}

func (f FileXattr) Patch(patches []webdav.Proppatch) ([]webdav.Propstat, error) {
	status := make([]webdav.Propstat, 0, len(patches))
	fstat, err := f.File.Stat()
	if err != nil {
		return nil, err
	}
	fName := fstat.Name()

	for _, patch := range patches {
		stat := webdav.Propstat{Props: patch.Props}
		if patch.Remove {
			success := true
			for _, prop := range patch.Props {
				attr := propertyToAttr(prop)
				err := xattr.Remove(fName, attr)
				if err != nil {
					success = false
					stat.ResponseDescription += fmt.Sprintf("attr: %v, err: %v", attr, err.Error())
				}
			}
			stat.Status = 200
			if !success {
				stat.Status = 500
			}
		} else {
			success := true
			for _, prop := range patch.Props {
				attr := propertyToAttr(prop)
				err := xattr.Set(fName, attr, prop.InnerXML)
				if err != nil {
					success = false
					stat.ResponseDescription += fmt.Sprintf("attr: %v, err: %v", attr, err.Error())
				}
			}
			stat.Status = 201
			if !success {
				stat.Status = 500
			}

		}

		status = append(status, stat)
	}

	return status, nil
}
