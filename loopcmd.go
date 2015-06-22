package main

import (
	"log"
	"os/exec"
	"time"
)

type LoopCmd struct {
	path string
	args []string
	cmd  *exec.Cmd

	stop chan struct{}
	next chan struct{}
}

func LoopCmdStart(filename string) (Stopper, error) {
	lc := &LoopCmd{
		path: "play",
		args: []string{filename},
		stop: make(chan struct{}, 3),
		next: make(chan struct{}, 3),
	}
	err := lc.start()
	go lc.run()
	return lc, err
}

func (lc *LoopCmd) Stop() {
	if lc == nil {
		return
	}
	close(lc.stop)
	if lc.cmd == nil || lc.cmd.Process == nil {
		return
	}
	lc.cmd.Process.Kill()
}

func (lc *LoopCmd) start() error {
	cmd := exec.Command(lc.path, lc.args...)
	lc.cmd = cmd
	err := cmd.Start()
	if err == nil {
		go lc.wait(cmd)
	}
	return err
}

func (lc *LoopCmd) wait(cmd *exec.Cmd) {
	cmd.Wait()
	lc.next <- struct{}{}
}

func (lc *LoopCmd) run() {
	tick := time.NewTicker(time.Millisecond * 15)
	for {
		select {
		case <-lc.next:
			err := lc.start()
			if err != nil {
				log.Print(err)
			}
		case <-lc.stop:
			tick.Stop()
			return
		}
	}
}
