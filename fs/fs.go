package fs

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/knight42/Yuki/common"
)

type Type byte

const (
	DEFAULT Type = iota
	XFS
	ZFS
)

type GetSizer interface {
	GetSize(string) int64
}

func New(ty Type) GetSizer {
	switch ty {
	case XFS:
		return &xfs{}
	case ZFS:
		return &zfs{}
	default:
		return &defaultFs{}
	}
	return &defaultFs{}
}

type defaultFs struct{}

func (f *defaultFs) GetSize(d string) int64 {
	return -1
}

type zfs struct{}

func (f *zfs) GetSize(d string) int64 {
	if !common.DirExists(d) {
		return -1
	}
	var buf bytes.Buffer
	cmd := exec.Command("df", "--output=used", d)
	cmd.Stdout = &buf
	if err := cmd.Run(); err != nil {
		return -1
	}
	scanner := bufio.NewScanner(&buf)
	scanner.Scan()
	scanner.Scan()
	kbs, err := strconv.ParseInt(scanner.Text(), 10, 64)
	if err != nil {
		return -1
	}
	return kbs
}

type xfs struct{}

func (f *xfs) GetSize(d string) int64 {
	if !common.DirExists(d) {
		return -1
	}

	var buf bytes.Buffer
	cmd := exec.Command("sudo", "xfs_quota", "-c", fmt.Sprintf("quota -pN %s", d))
	cmd.Stdout = &buf
	if err := cmd.Run(); err != nil {
		return -1
	}
	scanner := bufio.NewScanner(&buf)
	scanner.Scan()
	scanner.Scan()
	fields := strings.Fields(scanner.Text())
	kbs, err := strconv.ParseInt(fields[0], 10, 64)
	if err != nil {
		return -1
	}
	return kbs
}
