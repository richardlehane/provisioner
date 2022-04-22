package main

import (
	"errors"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"

	"github.com/andybalholm/cascadia"
	"github.com/packethost/packngo"
	"github.com/richardlehane/crock32"
)

var (
	dumpf  = flag.Bool("dump", false, "dump config file and quit (for debugging)")
	delf   = flag.Bool("delete", false, "delete server with host name -host")
	dcf    = flag.String("dc", "sjc1", "packet data centre location")
	slugf  = flag.String("slug", "c3.small.x86", "slug of machine type")
	osf    = flag.String("os", "ubuntu_20_04", "os type")
	pnamef = flag.String("project", "bench", "name of your packet project")
	hnamef = flag.String("host", "test.server", "host name for your new server")
	lifef  = flag.Duration("life", time.Hour, "duration before server is terminated")
	maxf   = flag.Float64("max", 0, "maximum price per hour. Give 0 to set at the on demand price. Give -1 to force on demand instance.")
	replf  = flag.String("replace", "", "comma-separated key-value pairs to replace ${KEY} strings in install")
	envf   = flag.String("env", "", "comma-separated list of environment variables to replace ${KEY} strings in install")
	filesf = flag.String("files", "", "comma-separated list of file names to replace ${KEY} strings in install")
	tuf    = flag.String("tu", "", "URL of html page with a date to calculate a throttle period")
	tsf    = flag.String("ts", "", "CSS selector for getting date in YYYY-MM-DD format to calculate a throttle period")
	tdf    = flag.String("td", "", "number of days to add to date in order to calculate a throttle period")
)

var stdPrices = map[string]float64{
	"c3.small.x86":  0.5,  // https://metal.equinix.com/product/servers/c3-small/ 8 cores @ 3.40 GHz, 32GB RAM, 960 GB SSD
	"c3.medium.x86": 1.1,  // https://metal.equinix.com/product/servers/c3-medium/ 24 cores @ 2.8 GHz, 64GB DDR4 RAM, 960 GB SSD
	"m3.small.x86":  1.05, // name: m3.small.x86 https://metal.equinix.com/product/servers/m3-small/ 8 cores @ 2.8 GHz, 64GB RAM, 960 GB SSD
	"m3.large.x86":  2,    // https://metal.equinix.com/product/servers/m3-large/ 32 cores @ 2.5 GHz, 256GB DDR4 RAM, 2 x 3.8 TB NVMe
	"s3.xlarge.x86": 1.85, // https://metal.equinix.com/product/servers/s3-xlarge/ 24 cores @ 2.2 GHz, 192GB DDR4 RAM, 1.9 TB SSD
}

func beefier(than string) []string {
	prc := stdPrices[than]
	ret := make([]string, 0, len(stdPrices))
	for k, v := range stdPrices {
		if v > prc {
			ret = append(ret, k)
		}
	}
	return ret
}

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
	// check if a throttle period is given
	terr := throttle(*tuf, *tsf, *tdf)
	if terr != nil {
		log.Fatalf("throtting: %s\n", terr)
	}
	var machine string
	plans, _, err := c.Plans.List(nil)
	if err != nil {
		log.Fatal(err)
	}
	for _, p := range plans {
		if p.Slug == *slugf {
			machine = p.Name
			break
		}
	}
	if machine == "" {
		log.Fatalf("Can't find slug %s in list of plans\n", *slugf)
	}
	if _, ok := stdPrices[machine]; !ok {
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
			*maxf = stdPrices[machine]
		}
		pri, _, err := c.SpotMarket.Prices()
		if err != nil {
			log.Fatal(err)
		}
		// if we bid the std price or more, and the spot is over that, try to upgrade
		if *maxf >= stdPrices[machine] && pri[*dcf][*slugf] >= *maxf {
			// try an upgrade
			machines := beefier(machine)
			slugs := make([]string, len(machines))
			for _, p := range plans {
				for i, mach := range machines {
					if p.Name == mach {
						slugs[i] = p.Slug
						break
					}
				}
			}
			bestPrice := stdPrices[machine]
			for idx, s := range slugs {
				if pri[*dcf][s] < bestPrice {
					bestPrice = pri[*dcf][s]
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
	dcr := provision(pid, host, plan, install, spot)
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

func provision(pid, host, plan, install string, spot bool) *packngo.DeviceCreateRequest {
	term := &packngo.Timestamp{Time: time.Now().Add(*lifef)}
	return &packngo.DeviceCreateRequest{
		Hostname:        host,
		Facility:        []string{*dcf},
		Plan:            plan,
		OS:              *osf,
		ProjectID:       pid,
		UserData:        install,
		BillingCycle:    "hourly",
		SpotInstance:    spot,
		SpotPriceMax:    *maxf,
		TerminationTime: term,
	}
}

func throttle(url, selector, duration string) error {
	if url == "" && selector == "" && duration == "" {
		return nil
	}
	if url == "" || selector == "" || duration == "" {
		return errors.New("must give url, selector and duration values")
	}
	days, err := strconv.Atoi(duration)
	if err != nil {
		return errors.New("invalid duration")
	}
	sel, err := cascadia.Compile(selector)
	if err != nil {
		return err
	}
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	node, err := html.Parse(resp.Body)
	if err != nil {
		return err
	}
	el := sel.MatchFirst(node)
	if el == nil || el.FirstChild == nil || len(el.FirstChild.Data) < 10 {
		return errors.New("selector found no node, empty node or no date")
	}
	t, err := time.Parse("2006-01-02", el.FirstChild.Data[:10])
	if err != nil {
		return err
	}
	next := t.AddDate(0, 0, days)
	if time.Now().After(next) {
		return nil
	}
	return errors.New("next run is " + next.String())
}
