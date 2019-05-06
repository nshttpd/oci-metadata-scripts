package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"oci-metadata-scripts/instancemeta"
	"github.com/oracle/oci-go-sdk/common/auth"
	"github.com/oracle/oci-go-sdk/objectstorage"
	log "github.com/sirupsen/logrus"
)

// FetchScripts will look in the instance metadata for any script attributes
// for the current phase (startup, shutdown) and attempt to gather them into
// a temporary working directory for execution.
func (sm *ScriptManager) FetchScripts() ([]string, error) {
	im, err := instancemeta.New().Get()
	if err != nil {
		return nil, fmt.Errorf("error fetching instance metadata : %s", err)
	}

	var scripts []string

	// run through the known metadata attributes for the script type and if they
	// are there handle retrieving them. If successful return the name that was
	// stored in the work dir to then execute.
	for _, a := range sm.attributes {
		attr := fmt.Sprintf("%s-%s", sm.Type.String(), a)
		if v, ok := im.Metadata[attr]; ok {
			log.Infof("Found %s in metadata : %s", attr, v)
			var g bool
			var sn string
			switch a {
			case "script":
				sn, g = handleScript(sm.WorkDir, v)
			case "script-url":
				sn, g = handleRemoteScript(sm.WorkDir, v)
			}
			if g {
				scripts = append(scripts, sn)
			}
		}
	}

	return scripts, nil
}

// handleScript will take the value supplied from the metadata for a *-script attribute
// and attempt to either decode it for a Base64 file or copy it from local file into
// the working directory for this invocation. If successful returns the script name
// to later be executed.
func handleScript(wd string, s string) (string, bool) {

	success := false
	var sName string
	var dName string

	// check to see if it's a local to host script
	if strings.HasPrefix(s, "/") {
		sName = filepath.Base(s)
		dName = fmt.Sprintf(wd + "/" + sName)
		// make sure we can stat the source file
		srcStat, err := os.Stat(s)
		if err != nil {
			log.Errorf("error reading source file '%s' : %s", s, err)
		} else {
			if !srcStat.Mode().IsRegular() {
				log.Errorf("source file '%s' is not a regular file", s)
			} else {
				src, err := os.Open(s)
				if err != nil {
					log.Errorf("error opening source file '%s' : %s", s, err)
				} else {
					defer src.Close()
					dest, err := os.Create(dName)
					if err != nil {
						log.Errorf("error opening destination '%s' : %s", dName, err)
					} else {
						defer dest.Close()
						n, err := io.Copy(dest, src)
						if err != nil {
							log.Errorf("error copying script file : %s", err)
						} else {
							log.Debugf("copied %d bytes of script file to work dir", n)
							success = true
						}
					}
				}
			}
		}
	} else { // if not local see if it's a base64 encoded script.
		src, err := base64.StdEncoding.DecodeString(s)
		if err != nil {
			log.Errorf("error decoding base64 data : %s", err)
		} else {
			dest, err := ioutil.TempFile(wd, "script")
			if err != nil {
				log.Errorf("error creating temp file : %s", err)
			} else {
				defer dest.Close()
				sName = filepath.Base(dest.Name())
				dName = dest.Name()
				n, err := dest.Write([]byte(src))
				if err != nil {
					log.Errorf("error writing to file '%s' : %s", dName, err)
				} else {
					log.Debugf("write %d bytes to script file '%s'", n, dName)
					success = true
				}
			}
		}
	}

	if success {
		if err := os.Chmod(dName, 0744); err != nil {
			log.Errorf("error on chmod of dest file '%s' : %s", dName, err)
			return "", false
		}
		return sName, success
	}
	return "", success
}

// handleRemoteScript will fetch the startup script from either a remote HTTP location
// or an OCI OSS storage bucket. It is the responsibility of the user to make sure these
// source locations are accessible to the instance.
func handleRemoteScript(wd string, s string) (string, bool) {
	success := false
	var sName string

	parts := strings.Split(s, "/")
	if len(parts) > 0 {
		sName = parts[len(parts)-1]
	}
	dName := fmt.Sprintf(wd + "/" + sName)

	// fetch from http(s) source and store.
	if strings.HasPrefix(s, "http") {
		resp, err := http.Get(s)
		if err != nil {
			log.Errorf("error fetching remote script '%s' : %s", s, err)
		} else {
			defer resp.Body.Close()
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Errorf("error reading body of remote script '%s': %s", s, err)
			} else {
				err := ioutil.WriteFile(dName, body, 0744)
				if err != nil {
					log.Errorf("error writing body of remote scripts '%s': %s", s, err)
				} else {
					success = true
				}
			}
		}
	}

	// fetch from OCI OSS bucket
	if strings.HasPrefix(s, "oci") {
		// regex out the bucket@namespace and file which could be not at root
		re := regexp.MustCompile(`([a-zA-Z0-9\-_]+)@([a-zA-Z0-9\-_]+)/(.*)`)
		parts := re.FindStringSubmatch(s)
		if len(parts) != 4 {
			log.Errorf("could not parse OCI info from request '%s'", s)
		} else {
			provider, err := auth.InstancePrincipalConfigurationProvider()
			oss, err := objectstorage.NewObjectStorageClientWithConfigurationProvider(provider)
			if err != nil {
				log.Errorf("error creating oci objectstorage client : %s", err)
			} else {
				req := objectstorage.GetObjectRequest{
					NamespaceName: &parts[2],
					BucketName:    &parts[1],
					ObjectName:    &parts[3],
				}
				res, err := oss.GetObject(context.Background(), req)
				if err != nil {
					log.Errorf("error fetching object from objectstorage '%s': %s", s, err)
				} else {
					buf, err := ioutil.ReadAll(res.Content)
					log.Debugf("bucket object size %d and read size %d", *res.ContentLength, len(buf))
					if err != nil || int64(len(buf)) != *res.ContentLength {
						log.Errorf("error reading data from object '%s': %s", s, err)
					} else {
						err := ioutil.WriteFile(dName, buf, 0744)
						if err != nil {
							log.Errorf("error writing body of remote script '%s': %s", s, err)
						} else {
							success = true
						}
					}
				}
			}
		}
	}

	return sName, success
}
