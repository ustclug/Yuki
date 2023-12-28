// Package fs implements function for getting the size of a given directory.
package fs

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"path"
	"strconv"
	"strings"

	"github.com/ustclug/Yuki/pkg/utils"
)

// Type represents different kinds of file system.
type Type byte

const (
	// DEFAULT represents the default file system. Always return -1 as size.
	DEFAULT Type = iota
	// XFS is the XFS file system. Getting the size by running `sudo -n xfs_quota -c "quota -pN $name"`.
	XFS
	// ZFS is the ZFS file system. Getting the size by running `df -B1 --output=used`.
	ZFS
)

// GetSizer is the interface that wraps the `GetSize` method.
type GetSizer interface {
	GetSize(string) int64
}

// New returns different `GetSizer` depending on the passed `Type`.
func New(ty Type) GetSizer {
	switch ty {
	case XFS:
		return &xfs{}
	case ZFS:
		return &zfs{}
	default:
		return &defaultFs{}
	}
}

type defaultFs struct{}

func (f *defaultFs) GetSize(d string) int64 {
	return -1
}

type zfs struct{}

func (f *zfs) GetSize(d string) int64 {
	if !utils.DirExists(d) {
		return -1
	}
	var buf bytes.Buffer
	cmd := exec.Command("df", "-B1", "--output=used", d)
	cmd.Stdout = &buf
	if err := cmd.Run(); err != nil {
		return -1
	}
	scanner := bufio.NewScanner(&buf)
	scanner.Scan()
	scanner.Scan()
	bs, err := strconv.ParseInt(scanner.Text(), 10, 64)
	if err != nil {
		return -1
	}
	return bs
}

type xfs struct{}

func (f *xfs) GetSize(d string) int64 {
	if !utils.DirExists(d) {
		return -1
	}

	var buf bytes.Buffer
	var kbs int64
	var err error
	name := path.Base(d)
	cmd := exec.Command("sudo", "-n", "xfs_quota", "-c", fmt.Sprintf("quota -pN %s", name))
	cmd.Stdout = &buf
	if err := cmd.Run(); err != nil {
		return -1
	}
	scanner := bufio.NewScanner(&buf)
	scanner.Scan()
	fields := strings.Fields(scanner.Text())
	switch {
	case len(fields) == 0:
		return -1
	case len(fields) >= 2:
		kbs, err = strconv.ParseInt(fields[1], 10, 64)
	default:
		scanner.Scan()
		fields = strings.Fields(scanner.Text())
		kbs, err = strconv.ParseInt(fields[0], 10, 64)
	}
	if err != nil {
		return -1
	}
	return kbs * 1024
}
