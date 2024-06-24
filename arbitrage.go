package main

import (
	"errors"
	"fmt"
	"math"
)

// arbitrage rules:
// if 0: return non-spot

func filterRegion(mp dcMachinePrices, reg string) dcMachinePrices {
	ret := make(dcMachinePrices)
	ret[reg] = mp[reg]
	return ret
}

func listPlans(st stdPrices) []string {
	var idx int
	ret := make([]string, len(st))
	for k := range st {
		ret[idx] = k
		idx++
	}
	return ret
}

func filterPlans(mp dcMachinePrices, plans []string) dcMachinePrices {
	ret := make(dcMachinePrices)
	for _, plan := range plans {
		for reg, m := range mp {
			if _, ok := m[plan]; !ok {
				continue
			}
			if _, ok := ret[reg]; !ok {
				ret[reg] = make(map[string]float32)
			}
			ret[reg][plan] = m[plan]
		}
	}
	return ret
}

func arbitrage(c client, std stdPrices, max float32) (region string, plan string, price float32, spot bool, err error) {
	// if max is 0 we expect a slug arg and return prices and region for that
	if max == 0 {
		if pri, ok := std[*slugf]; ok {
			plan = *slugf
			price = pri
			// if region/ data centre flag set
			if *dcf != "" {
				region = *dcf
				return
			}
			// otherwise set region to default for client
			switch c.(type) {
			case *cherryClient:
				region = cherryDC
			case *equinixClient:
				region = equinixMetro
			}
			return
		}
		err = fmt.Errorf("slug %s not found in cherry or equinix plans", *slugf)
		return
	}
	// get prices from the client
	pri, e := c.Prices()
	if e != nil {
		err = e
		return
	}
	// apply filters to prices
	if *dcf != "" {
		pri = filterRegion(pri, *dcf)
	}
	if *slugf != "" {
		pri = filterPlans(pri, []string{*slugf})
	}
	// if max negative, go for cheapest
	if max < 0 {
		var curr float32
		for reg, m := range pri {
			for pla, p := range m {
				n := float32(math.Abs(float64(p)))
				if curr == 0 || n < curr {
					region = reg
					plan = pla
					price = n
					spot = p > 0
					curr = n
				}
			}
		}
		if curr == 0 {
			err = errors.New("no prices found")
		}
		return
	}
	// if max postive, go for best value
	var val float32
	var curr float32
	for reg, m := range pri {
		for pla, p := range m {
			n := float32(math.Abs(float64(p)))
			val = std[pla] - n
			if (n <= max && curr == 0) || (n <= max && val > curr) {
				region = reg
				plan = pla
				price = n
				spot = p > 0
				curr = val
			}
		}
	}
	if curr == 0 {
		err = fmt.Errorf("no prices found cheaper than max %v", max)
	}
	return
}
