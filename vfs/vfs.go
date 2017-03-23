package vfs

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/yasker/backupstore"
	"github.com/yasker/backupstore/util"
)

var (
	log = logrus.WithFields(logrus.Fields{"pkg": "vfs"})
)

type BackupStoreDriver struct {
	destURL string
	path    string
}

const (
	KIND = "vfs"

	VfsPath = "vfs.path"

	MaxCleanupLevel = 10
)

func init() {
	if err := backupstore.RegisterDriver(KIND, initFunc); err != nil {
		panic(err)
	}
}

func initFunc(destURL string) (backupstore.BackupStoreDriver, error) {
	b := &BackupStoreDriver{}
	u, err := url.Parse(destURL)
	if err != nil {
		return nil, err
	}

	if u.Scheme != KIND {
		return nil, fmt.Errorf("BUG: Why dispatch %v to %v?", u.Scheme, KIND)
	}

	if u.Host != "" {
		return nil, fmt.Errorf("VFS path must follow: vfs:///path/ format")
	}

	b.path = u.Path

	if b.path == "" {
		return nil, fmt.Errorf("Cannot find vfs path")
	}
	if _, err := b.List(""); err != nil {
		return nil, fmt.Errorf("VFS path %v doesn't exist or is not a directory", b.path)
	}

	b.destURL = KIND + "://" + b.path
	log.Debug("Loaded driver for %v", b.destURL)
	return b, nil
}

func (v *BackupStoreDriver) updatePath(path string) string {
	return filepath.Join(v.path, path)
}

func (v *BackupStoreDriver) preparePath(file string) error {
	if err := os.MkdirAll(filepath.Dir(v.updatePath(file)), os.ModeDir|0700); err != nil {
		return err
	}
	return nil
}

func (v *BackupStoreDriver) Kind() string {
	return KIND
}

func (v *BackupStoreDriver) GetURL() string {
	return v.destURL
}

func (v *BackupStoreDriver) FileSize(filePath string) int64 {
	file := v.updatePath(filePath)
	st, err := os.Stat(file)
	if err != nil || st.IsDir() {
		return -1
	}
	return st.Size()
}

func (v *BackupStoreDriver) FileExists(filePath string) bool {
	return v.FileSize(filePath) >= 0
}

func (v *BackupStoreDriver) Remove(names ...string) error {
	for _, name := range names {
		if err := os.RemoveAll(v.updatePath(name)); err != nil {
			return err
		}
		//Also automatically cleanup upper level directories
		dir := v.updatePath(name)
		for i := 0; i < MaxCleanupLevel; i++ {
			dir = filepath.Dir(dir)
			// Don't clean above backupstore base
			if strings.HasSuffix(dir, backupstore.GetBackupstoreBase()) {
				break
			}
			// If directory is not empty, then we don't need to continue
			if err := os.Remove(dir); err != nil {
				break
			}
		}
	}
	return nil
}

func (v *BackupStoreDriver) Read(src string) (io.ReadCloser, error) {
	file, err := os.Open(v.updatePath(src))
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (v *BackupStoreDriver) Write(dst string, rs io.ReadSeeker) error {
	tmpFile := dst + ".tmp"
	if v.FileExists(tmpFile) {
		v.Remove(tmpFile)
	}
	if err := v.preparePath(dst); err != nil {
		return err
	}
	file, err := os.Create(v.updatePath(tmpFile))
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(file, rs)
	if err != nil {
		return err
	}

	if v.FileExists(dst) {
		v.Remove(dst)
	}
	return os.Rename(v.updatePath(tmpFile), v.updatePath(dst))
}

func (v *BackupStoreDriver) List(path string) ([]string, error) {
	out, err := util.Execute("ls", []string{"-1", v.updatePath(path)})
	if err != nil {
		return nil, err
	}
	var result []string
	if len(out) == 0 {
		return result, nil
	}
	result = strings.Split(strings.TrimSpace(string(out)), "\n")
	return result, nil
}

func (v *BackupStoreDriver) Upload(src, dst string) error {
	tmpDst := dst + ".tmp"
	if v.FileExists(tmpDst) {
		v.Remove(tmpDst)
	}
	if err := v.preparePath(dst); err != nil {
		return err
	}
	_, err := util.Execute("cp", []string{src, v.updatePath(tmpDst)})
	if err != nil {
		return err
	}
	_, err = util.Execute("mv", []string{v.updatePath(tmpDst), v.updatePath(dst)})
	if err != nil {
		return err
	}
	return nil
}

func (v *BackupStoreDriver) Download(src, dst string) error {
	_, err := util.Execute("cp", []string{v.updatePath(src), dst})
	if err != nil {
		return err
	}
	return nil
}
