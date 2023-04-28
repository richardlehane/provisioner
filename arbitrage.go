package main

/*
import (
	"log"
	"strings"

	"github.com/richardlehane/crock32"
)

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

func arbitrage() {

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
}
*/
