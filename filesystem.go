package webdav

import (
	"context"
	"encoding/xml"
	"strings"

	"os"

	xattr "github.com/pkg/xattr"
	webdav "golang.org/x/net/webdav"
)

var _ webdav.FileSystem = (*WrapFS)(nil)

type WrapFS struct {
	fileSystem webdav.Dir
}

func (fs *WrapFS) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
	return fs.fileSystem.Mkdir(ctx, name, perm)
}
func (fs *WrapFS) OpenFile(ctx context.Context, name string, flag int, perm os.FileMode) (webdav.File, error) {
	file, err := fs.fileSystem.OpenFile(ctx, name, flag, perm)
	wrapped := FileXattr{file}
	return wrapped, err
}
func (fs *WrapFS) RemoveAll(ctx context.Context, name string) error {
	return fs.fileSystem.RemoveAll(ctx, name)
}
func (fs *WrapFS) Rename(ctx context.Context, oldName, newName string) error {
	return fs.fileSystem.Rename(ctx, oldName, newName)
}
func (fs *WrapFS) Stat(ctx context.Context, name string) (os.FileInfo, error) {
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

	return xml.Name{Space: namespace, Local: localName}, true
}

func (f *FileXattr) DeadProps() (map[xml.Name]webdav.Property, error) {
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

func (f *FileXattr) Patch([]webdav.Proppatch) ([]webdav.Propstat, error) {
	return nil, nil
}
