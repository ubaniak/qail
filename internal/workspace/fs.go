package workspace

import "os"

// FS is the narrow filesystem interface workspace methods use to mkdir,
// remove, stat, and list entries. The default OSFS adapter wraps the os
// package; tests substitute a fake to exercise Create / Remove /
// RemoveRepo / Clean without touching disk.
//
// The interface lives in the consumer (workspace) so tests can fake it
// without an extra package; only the operations workspace actually
// performs are exposed.
type FS interface {
	Stat(name string) (os.FileInfo, error)
	Mkdir(path string, perm os.FileMode) error
	RemoveAll(path string) error
	ReadDir(name string) ([]os.DirEntry, error)
}

// OSFS is the production FS adapter, delegating to the os package.
type OSFS struct{}

func (OSFS) Stat(name string) (os.FileInfo, error)      { return os.Stat(name) }
func (OSFS) Mkdir(path string, perm os.FileMode) error  { return os.Mkdir(path, perm) }
func (OSFS) RemoveAll(path string) error                { return os.RemoveAll(path) }
func (OSFS) ReadDir(name string) ([]os.DirEntry, error) { return os.ReadDir(name) }
