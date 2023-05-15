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
	dryf   = flag.Bool("dry", false, "do a dry run of a host provisioning")
	dcf    = flag.String("dc", "", "slug of region/ data centre")
	slugf  = flag.String("slug", "", "slug of machine/ plan")
	osf    = flag.String("os", "", "os type")
	pnamef = flag.String("project", "bench", "project name")
	hnamef = flag.String("host", "test.server", "server host name")
	lifef  = flag.Duration("life", time.Hour, "duration before server is terminated (doesn't work for cherry)")
	maxf   = flag.Float64("max", 0.15, "maximum price per hour. If positive, best spot instance up to the price will be selected. If 0, on demand price for -slug. If negative, cheapest spot instance below the abs price.")
	replf  = flag.String("replace", "", "comma-separated key-value pairs to replace ${KEY} strings in install")
	envf   = flag.String("env", "", "comma-separated list of environment variables to replace ${KEY} strings in install")
	filesf = flag.String("files", "", "comma-separated list of file names to replace ${KEY} strings in install")
)

type stdPrices map[string]float64 // machine[standard hourly price in USD]

type dcMachinePrices map[string]map[string]float64 //region[machine[price (positive for spot, negative for standard)]]

type client interface {
	Provision(host, install, dc, plan string, price float64, spot bool) error
	Delete(host string) error
	// Informational
	Facilities() ([][2]string, error)
	Machines() ([][2]string, error)
	OSs() ([][2]string, error)
	Prices() (dcMachinePrices, error)
}

func main() {
	flag.Parse()
	// get a client
	var c client
	var std stdPrices
	var err error
	if *slugf == "" {
		cClient, err := cherry(*pnamef)
		if err != nil {
			log.Fatal(err)
		}
		eClient, err := equinix(*pnamef)
		if err != nil {
			log.Fatal(err)
		}
		std = joinStd(cherryPlans, equinixPlans)
		c = &joint{[]client{cClient, eClient}}
	} else {
		if _, ok := cherryPlans[*slugf]; ok {
			c, err = cherry(*pnamef)
			std = cherryPlans
			if err != nil {
				log.Fatal(err)
			}
		} else {
			if _, ok := equinixPlans[*slugf]; ok {
				c, err = equinix(*pnamef)
				std = equinixPlans
				if err != nil {
					log.Fatal(err)
				}
			} else {
				log.Printf("can't find slug: %s", *slugf)
				os.Exit(1)
			}
		}
	}
	// if we're deleting...
	if *delf {
		if err := c.Delete(*hnamef); err != nil {
			log.Fatal(err)
		}
		log.Printf("deleted host %s", *hnamef)
		return
	}
	// arbitrage
	host := strings.Replace(*hnamef, "RAND", crock32.PUID(), -1)
	dc, machine, pri, spot, err := arbitrage(c, std, *maxf)
	if err != nil {
		log.Fatal(err)
	}
	// populate install
	install := readInstall(flag.Arg(0), host, machine)
	if *dumpf {
		log.Print(install)
		return
	}
	// dry run
	if *dryf {
		log.Printf("dry run: provisioning...\nhost: %s\ndc: %s\nmachine: %s\nprice: %v\nspot: %v\n", host, dc, machine, pri, spot)
		return
	}
	// provision
	if err := c.Provision(host, install, dc, machine, pri, spot); err != nil {
		log.Fatal(sanitize(err.Error()))
	}
	log.Printf("provisioned...\nhost: %s\ndc: %s\nmachine: %s\nprice: %v\nspot: %v\n", host, dc, machine, pri, spot)
}

func replaceStrs() []string {
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
	return vals
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
		vals := replaceStrs()
		repl := strings.NewReplacer(vals...)
		install = repl.Replace(install)
	}
	return install
}

func sanitize(in string) string {
	sstr := "XXXXXXX"
	if *replf != "" || *envf != "" || *filesf != "" {
		vals := replaceStrs()
		if len(vals) < 2 {
			return in
		}
		vals = vals[1:]
		var odd bool
		for i := range vals {
			if odd {
				odd = false
				vals[i] = sstr
				continue
			}
			odd = true
		}
		vals = append(vals, sstr)
		repl := strings.NewReplacer(vals...)
		in = repl.Replace(in)
	}
	return in
}
