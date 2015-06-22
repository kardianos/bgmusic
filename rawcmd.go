package main

import (
	"io/ioutil"
	"os/exec"
	"strings"
)

type RawCmd struct {
	name string
	cmd  *exec.Cmd
	from []byte
	size uint64

	virtualPosition uint64
}

func RawCmdStart(name string) (Stopper, error) {
	file, err := ioutil.ReadFile(name)
	if err != nil {
		return nil, err
	}

	cmd := exec.Command("play", strings.Split("-q -t raw -e signed-integer -b 16 --endian little -c 1 -r 48000 -", " ")...)
	rc := &RawCmd{
		name: name,
		cmd:  cmd,
		from: file,
		size: uint64(len(file)),
	}
	cmd.Stdin = rc
	return rc, cmd.Start()
}

func (rc *RawCmd) Read(buf []byte) (int, error) {
	pos := int(rc.virtualPosition % rc.size)
	n := copy(buf, rc.from[pos:])
	rc.virtualPosition += uint64(n)
	return n, nil
}

func (rc *RawCmd) Stop() {
	if rc == nil {
		return
	}
	if rc.cmd != nil && rc.cmd.Process != nil {
		rc.cmd.Process.Kill()
	}
}
