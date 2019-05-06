package main

import (
	"flag"
	"io/ioutil"
	"os"

	log "github.com/sirupsen/logrus"
)

var (
	debug      bool
	scriptType string
)

const (
	workDir   = "/var/lib/oci"
	workShell = "/bin/bash"
)

type ScriptManager struct {
	Type       ScriptType
	WorkDir    string
	WorkShell  string
	attributes [2]string
}

func main() {
	flag.BoolVar(&debug, "debug", false, "display debug information")
	flag.StringVar(&scriptType, "script-type", "unknown", "script type to run")
	flag.Parse()

	if debug {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	st, err := ParseScriptType(scriptType)
	if err != nil {
		log.Error("No valid argument specified for script type")
		log.Error("%s : %s", err, scriptType)
		os.Exit(1)
	}

	log.Infof("starting %s scripts", st)

	wd, err := ioutil.TempDir(workDir, st.String()+"-")
	if err != nil {
		log.Errorf("error creating temp work directory : %s", err)
		os.Exit(1)
	}

	defer func() {
		if !debug {
			os.RemoveAll(wd)
		} else {
			log.Debug("not removing work dir : ", wd)
		}
	}()

	log.Debugf("temp work directory is : %s", wd)

	mgr := &ScriptManager{
		Type:       st,
		WorkDir:    wd,
		WorkShell:  workShell,
		attributes: [...]string{"script-url", "script"},
	}

	scripts, err := mgr.FetchScripts()

	if err != nil {
		log.Errorf("error retrieving %s scripts : %s", st, err)
		os.Exit(1)
	}

	for _, s := range scripts {
		err := mgr.RunScript(s)
		if err != nil {
			log.Errorf("%s", err)
		}
	}

	log.Infof("Finished running %s scripts", st)

}
