package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/richardlehane/crock32"
)

var (
	dumpf  = flag.Bool("dump", false, "dump config file and quit (for debugging)")
	delf   = flag.Bool("delete", false, "delete server with host name -host")
	dcf    = flag.String("dc", "", "data centre location/ region")
	slugf  = flag.String("slug", "", "slug of machine/ plan")
	osf    = flag.String("os", "", "os type")
	pnamef = flag.String("project", "bench", "project name")
	hnamef = flag.String("host", "test.server", "server host name")
	lifef  = flag.Duration("life", time.Hour, "duration before server is terminated (doesn't work for cherry)")
	maxf   = flag.Float64("max", 0, "maximum price per hour. Give 0 to set at the on demand price. Give -1 to force on demand instance.")
	replf  = flag.String("replace", "", "comma-separated key-value pairs to replace ${KEY} strings in install")
	envf   = flag.String("env", "", "comma-separated list of environment variables to replace ${KEY} strings in install")
	filesf = flag.String("files", "", "comma-separated list of file names to replace ${KEY} strings in install")
)

type stdPrices map[string]float64

type dcMachinePrices map[string]map[string]float64

type client interface {
	Provision(host, install string) error
	Delete(host string) error
	Arbitrage(max float64) (string, string, float64, bool) // region/ machine / price / bool
	// setters
	SetPlan(plan string) bool
	SetDC(dc string) bool
	SetSpot(spot bool)

	// Informational
	Facilities() ([][2]string, error)
	Machines() ([][2]string, error)
	OSs() ([][2]string, error)
	Prices() (dcMachinePrices, error)
}

func checkPlan(c client, p string) bool {
	plans, err := c.Machines()
	if err != nil {
		return false
	}
	for _, plan := range plans {
		if plan[0] == p {
			return true
		}
	}
	return false
}

func main() {
	flag.Parse()
	// Get a client
	c, err := equinix(*pnamef)
	if err != nil {
		log.Fatal(err)
	}
	// if we're deleting...
	if *delf {
		log.Print(c.Delete(*hnamef))
		return
	}
	// arbitrage
	host := strings.Replace(*hnamef, "RAND", crock32.PUID(), -1)
	dc, machine, _, spot := c.Arbitrage(*maxf)
	c.SetDC(dc)
	c.SetPlan(machine)
	c.SetSpot(spot)
	// populate install
	install := readInstall(flag.Arg(0), host, machine)
	if *dumpf {
		log.Print(install)
		return
	}
	// provision
	log.Print(c.Provision(host, install))
}

func readInstall(path, host, machine string) string {
	var install string
	if path == "" {
		return install
	}
	byt, err := os.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	install = string(byt)
	install = strings.Replace(install, "\r\n", "\n", -1)
	install = strings.Replace(install, "${HOST}", host, -1)
	install = strings.Replace(install, "${PROJECT}", *pnamef, -1)
	install = strings.Replace(install, "${MACHINE}", machine, -1)
	if *replf != "" || *envf != "" || *filesf != "" {
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
				byt, err := os.ReadFile(k)
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
