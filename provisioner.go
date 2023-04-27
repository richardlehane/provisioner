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
	svcf   = flag.String("service", "equinix", "metal provider")
	dcf    = flag.String("dc", "sv15", "data centre location/ region")
	slugf  = flag.String("slug", "m3.small.x86", "slug of machine/ plan")
	osf    = flag.String("os", "ubuntu_22_04", "os type")
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
	Provision(host, plan, install string, spot bool) error
	Delete(host string) error

	// Informational
	Facilities() ([][2]string, error)
	Machines() ([][2]string, error)
	OSs() ([][2]string, error)
	Prices() (dcMachinePrices, error)
}

func beefier(than string, stdp stdPrices) []string {
	prc := stdp[than]
	ret := make([]string, 0, len(stdp))
	for k, v := range stdp {
		if v > prc {
			ret = append(ret, k)
		}
	}
	return ret
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
	var machine string
	plans, err := c.Machines()
	if err != nil {
		log.Fatal(err)
	}
	for _, p := range plans {
		if p[0] == *slugf {
			machine = p[1]
			break
		}
	}
	if machine == "" {
		log.Fatalf("Can't find slug %s in list of plans\n", *slugf)
	}
	if _, ok := equinixPlans[machine]; !ok {
		log.Fatalf("Don't have a price for machine type %s\n", machine)
	}
	host := strings.Replace(*hnamef, "RAND", crock32.PUID(), -1)
	// now price arbitrage
	plan := *slugf
	spot := true
	// get an on demand instance
	if *maxf < 0 {
		spot = false
		*maxf = 0
	} else {
		// if max is set to 0, set it to the std on demand price
		if *maxf == 0 {
			*maxf = equinixPlans[machine]
		}
		pri, err := c.Prices()
		if err != nil {
			log.Fatal(err)
		}
		// if we bid the std price or more, and the spot is over that, try to upgrade
		if *maxf >= equinixPlans[machine] && pri["sv15"][*slugf] >= *maxf {
			// try an upgrade
			machines := beefier(machine, equinixPlans)
			slugs := make([]string, len(machines))
			for _, p := range plans {
				for i, mach := range machines {
					if p[1] == mach {
						slugs[i] = p[1]
						break
					}
				}
			}
			bestPrice := equinixPlans[machine]
			for idx, s := range slugs {
				if pri["sv15"][s] > 0 && pri["sv15"][s] < bestPrice {
					bestPrice = pri["sv15"][s]
					plan = s
					machine = machines[idx]
					*maxf = bestPrice
				}
			}
			// if we haven't upgraded, just get an on demand instance
			if plan == *slugf {
				spot = false
				*maxf = 0
			}
		}
	}
	// populate install
	install := readInstall(flag.Arg(0), host, machine)
	if *dumpf {
		log.Print(install)
		return
	}
	// provision
	log.Print(c.Provision(host, plan, install, spot))
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
