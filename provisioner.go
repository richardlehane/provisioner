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

var stdPrices = map[string]float64{
	"t1.small.x86":  0.07, // Tiny		Intel Atom C2550	x86	4 Cores @ 2.4 GHz	8 GB	80 GB
	"m2.xlarge.x86": 2,    // Memory	Intel Xeon Gold 5120 (2x)	x86	28 Cores @ 2.2 GHz	384 GB
	"c1.small.x86":  0.4,  // Compute	Intel Xeon E3-1240	x86	4 Cores @ 3.5 GHz	32 GB	120 GB
	"c2.medium.x86": 1,    // Compute	AMD EPYC 7401p	x86	24 Cores @ 2.2 GHz	64 GB	960 GB
	"m1.xlarge.x86": 1.7,  // Memory	Intel Xeon E5-2650 (2x)	x86	24 Cores @ 2.2 GHz	256 GB	2.8 TB
	"c1.large.arm":  0.5,  // Compute	Cavium ThunderX (2x)	Armv8	96 Cores @ 2.0 GHz	128 GB	250 GB
	"x1.small.x86":  0.4,  // Accelerator	Intel Xeon E3-1578L	x86	4 Cores x 2.0 GHz	32 GB	240 GB
	"c1.xlarge.x86": 1.75, // Compute	Intel Xeon E5-2640 (2x)	x86	16 Cores @ 2.6 GHz	128 GB	1.6 TB NVMe
	"s1.large.x86":  1.5,  // Storage	Intel Xeon E5-2620 (2x)	x86	16 Cores @ 2.1 GHz	128 GB	24 TB
}

var (
	dumpf  = flag.Bool("dump", false, "dump config file and quit (for debugging)")
	delf   = flag.Bool("delete", false, "delete server with host name -host")
	dcf    = flag.String("dc", "sjc1", "packet data centre location")
	slugf  = flag.String("slug", "baremetal_0", "slug of machine type")
	osf    = flag.String("os", "ubuntu_18_04", "os type")
	pnamef = flag.String("project", "bench", "name of your packet project")
	hnamef = flag.String("host", "test.server", "host name for your new server")
	lifef  = flag.Duration("life", time.Hour, "duration before server is terminated")
	maxf   = flag.Float64("max", 0, "maximum price per hour. Give 0 to set at the on demand price. Give -1 to force on demand instance.")
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
	ps, _, err := c.Projects.List(nil)
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
		log.Fatalf("Can't find project name %s\n", *pnamef)
	}
	// if we're deleting...
	if *delf {
		devices, _, err := c.Devices.List(pid, nil)
		if err != nil {
			log.Fatal(err)
		}
		var did string
		for _, d := range devices {
			if d.Hostname == *hnamef {
				did = d.ID
				break
			}
		}
		if did == "" {
			log.Fatalf("Can't find hostname %s in project %s\n", *hnamef, *pnamef)
		}
		_, err = c.Devices.Delete(did, true)
		log.Print(err)
		return
	}
	var machine string
	plans, _, err := c.Plans.List(nil)
	if err != nil {
		log.Fatal(err)
	}
	for _, p := range plans {
		if p.Slug == *slugf {
			machine = p.Name
		}
	}
	if machine == "" {
		log.Fatalf("Can't find slug %s in list of plans\n", *slugf)
	}
	if _, ok := stdPrices[machine]; !ok {
		log.Fatalf("Don't have a price for machine type %s\n", machine)
	}
	host := strings.Replace(*hnamef, "RAND", crock32.PUID(), -1)
	install := readInstall(flag.Arg(0), host, machine)
	if *dumpf {
		log.Print(install)
		return
	}
	// now price arbitrage
	spot := true
	// get an on demand instance
	if *maxf < 0 {
		spot = false
		*maxf = 0
	} else {
		// if max is set to 0, set it to the std on demand price
		if *maxf == 0 {
			*maxf = stdPrices[machine]
		}
		pri, _, err := c.SpotMarket.Prices()
		if err != nil {
			log.Fatal(err)
		}
		// if we bid the std price or more, and the spot is over that, then just get an on demand instance
		if *maxf >= stdPrices[machine] && pri[*dcf][*slugf] >= *maxf {
			spot = false
			*maxf = 0
		}
	}
	dcr := provision(pid, host, install, spot)
	_, _, err = c.Devices.Create(dcr)
	log.Print(err)
}

func readInstall(path, host, machine string) string {
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

func provision(pid, host, install string, spot bool) *packngo.DeviceCreateRequest {
	term := &packngo.Timestamp{Time: time.Now().Add(*lifef)}
	return &packngo.DeviceCreateRequest{
		Hostname:        host,
		Facility:        []string{*dcf},
		Plan:            *slugf,
		OS:              *osf,
		ProjectID:       pid,
		UserData:        install,
		BillingCycle:    "hourly",
		SpotInstance:    spot,
		SpotPriceMax:    *maxf,
		TerminationTime: term,
	}
}
