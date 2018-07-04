package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/packethost/packngo"
	"github.com/richardlehane/crock32"
)

var (
	pnamef = flag.String("project", "bench", "name of your packet project")
	hnamef = flag.String("host", "test.server", "host name for your new server")
	lifef  = flag.Duration("life", time.Hour, "duration before server is terminated")
	maxf   = flag.Float64("max", 0.07, "maximum price per hour")
	replf  = flag.String("replace", "", "comma-separated key-value pairs to replace ${KEY} strings in install")
	envf   = flag.String("env", "", "comma-separated list of environment variables to replace ${KEY} strings in install")
	filesf = flag.String("files", "", "comma-separated list of file names to replace ${KEY} strings in install")
)

func main() {
	flag.Parse()
	c, err := packngo.NewClient()
	if err != nil {
		log.Fatal(err)
	}
	ps, _, err := c.Projects.List()
	if err != nil {
		log.Fatal(err)
	}
	var pid string
	for _, p := range ps {
		if *pnamef == p.Name {
			pid = p.ID
			break
		}
	}
	if pid == "" {
		log.Fatal("Can't find project name")
	}
	install := readInstall(flag.Arg(0))
	dcr := provision(pid, install)
	_, _, err = c.Devices.Create(dcr)
	log.Print(err)
}

func readInstall(path string) string {
	var install string
	if path == "" {
		return install
	}
	byt, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	install = string(byt)
	install = strings.Replace(install, "\r\n", "\n", -1)
	if *replf != "" || *envf != "" {
		var vals []string
		if *replf != "" {
			vals = strings.Split(*replf, ",")
		}
		if *envf != "" {
			envs := strings.Split(*envf, ",")
			for _, k := range envs {
				v := os.Getenv(k)
				if v == "" {
					log.Fatalf("Can't find env key: %s", k)
				}
				vals = append(vals, k, v)
			}
		}
		if *filesf != "" {
			files := strings.Split(*filesf, ",")
			for _, k := range files {
				byt, err := ioutil.ReadFile(k)
				if err != nil {
					log.Fatalf("Can't open file: %s", k)
				}
				k = filepath.Base(k)
				vals = append(vals, k, string(byt))
			}
		}
		var odd bool
		for i, v := range vals {
			if odd {
				odd = false
				continue
			}
			vals[i] = "${" + v + "}"
			odd = true
		}
		repl := strings.NewReplacer(vals...)
		install = repl.Replace(install)
	}
	return install
}

func provision(pid, install string) *packngo.DeviceCreateRequest {
	term := &packngo.Timestamp{Time: time.Now().Add(*lifef)}
	return &packngo.DeviceCreateRequest{
		Hostname:        strings.Replace(*hnamef, "${RAND}", crock32.PUID()),
		Facility:        "sjc1",
		Plan:            "baremetal_0",
		OS:              "ubuntu_18_04",
		ProjectID:       pid,
		UserData:        install,
		BillingCycle:    "hourly",
		SpotInstance:    true,
		SpotPriceMax:    *maxf,
		TerminationTime: term,
	}
}
