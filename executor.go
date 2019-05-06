package main

import (
	"bufio"
	"fmt"
	"os/exec"

	log "github.com/sirupsen/logrus"
)

const defaultShell = "/bin/bash"

func (sm *ScriptManager) RunScript(sn string) error {

	s := fmt.Sprintf("%s/%s", sm.WorkDir, sn)

	log.Debugf("about to run : %s", s)

	cmd := exec.Command(defaultShell, s)

	cmdReader, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("error creating stdout pipe for '%s' : %s", s, err)
	}

	scanner := bufio.NewScanner(cmdReader)

	go func() {
		for scanner.Scan() {
			log.Infof("%s | %s", sn, scanner.Text())
		}
	}()

	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("error starting command '%s' : %s", s, err)
	}

	err = cmd.Wait()
	if err != nil {
		return fmt.Errorf("error waiting for command '%s': %s", s, err)
	}

	return nil
}
