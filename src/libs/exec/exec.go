package exec

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"
)

type Exec struct {
	path string
	args []string

	cmd   *exec.Cmd
	cmdMu sync.RWMutex
}

func NewExec(path string, args ...string) Exec {
	return Exec{
		path: path,
		args: args,
	}
}

func (e *Exec) startCmd() error {
	e.cmdMu.Lock()
	defer e.cmdMu.Unlock()

	if e.cmd != nil {
		return fmt.Errorf("exec cmd exists")
	}

	e.cmd = exec.Command(
		e.path,
		e.args...,
	)

	e.cmd.Stdout = os.Stdout
	e.cmd.Stderr = os.Stderr

	// start
	err := e.cmd.Start()
	if err != nil {
		e.cmd = nil
		return err
	}

	return nil
}

func (e *Exec) stopCmd() error {
	e.cmdMu.Lock()
	defer e.cmdMu.Unlock()

	if e.cmd == nil {
		return fmt.Errorf("exec null cmd")
	}

	// interrupt
	err := e.cmd.Process.Signal(os.Interrupt)
	if err != nil {
		// kill
		err = e.cmd.Process.Kill()
	} else {
		// wait end
		e.cmd.Wait()
	}
	e.cmd = nil

	return err
}

func (e *Exec) Start() error {
	return e.startCmd()
}

func (e *Exec) Stop() {
	err := e.stopCmd()
	if err != nil {
		log.Println("exec stop error", e.path, err)
	}
}
