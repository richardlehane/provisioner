package main

import (
	"log"

	"github.com/cherryservers/cherrygo/v3"
)

var cherryPlans = stdPrices{
	"e3_1240v3":  0.188, // https://www.cherryservers.com/pricing/dedicated-servers/e3_1240v3?b=37&r=1 4 cores @ 3.4GHz, 16GB ECC DDR3 RAM, 2x SSD 250GB
	"e3_1240v5":  0.197, // https://www.cherryservers.com/pricing/dedicated-servers/e5_1620v4?b=37&r=1 4 cores @ 3.5GHz, 32GB ECC DDR4 RAM, 2x SSD 250GB
	"e3_1240lv5": 0.197, // https://www.cherryservers.com/pricing/dedicated-servers/e3_1240v5?b=37&r=1 4 cores @ 3.5GHz, 32GB ECC DDR4 RAM, 2x SSD 250GB
	"e5_1620v4":  0.197, // https://www.cherryservers.com/pricing/dedicated-servers/e3_1240lv5?b=37&r=1 4 cores @ 2.1GHz, 32GB ECC DDR4 RAM, 2x SSD 250GB
}

type cherryClient struct {
	teamID int
	*cherrygo.Client
}

func (cc *cherryClient) Provision(host, plan, install string, spot bool) error {
	return nil
}

func (cc *cherryClient) Delete(host string) error {
	return nil
}

func (cc *cherryClient) Facilities() ([][2]string, error) {
	fac, _, err := cc.Regions.List(nil)
	if err != nil {
		return nil, err
	}
	ret := make([][2]string, len(fac))
	for idx, v := range fac {
		ret[idx][0], ret[idx][1] = v.Slug, v.Name
	}
	return ret, nil
}
func (cc *cherryClient) Machines() ([][2]string, error) {
	pla, _, err := cc.Plans.List(cc.teamID, nil)
	if err != nil {
		return nil, err
	}
	ret := make([][2]string, len(pla))
	for idx, v := range pla {
		ret[idx][0], ret[idx][1] = v.Slug, v.Name
	}
	return ret, nil
}

func (cc *cherryClient) Prices() (dcMachinePrices, error) {
	pla, _, err := cc.Plans.List(cc.teamID, nil)
	if err != nil {
		return nil, err
	}
	ret := make(dcMachinePrices)
	for _, plan := range pla {
		if _, ok := cherryPlans[plan.Slug]; !ok {
			continue
		}
		for _, reg := range plan.AvailableRegions {
			//if reg.SpotQty < 1 {
			//	continue
			//}
			if _, ok := ret[reg.Slug]; !ok {
				ret[reg.Slug] = make(map[string]float64)
			}
			for idx, pri := range plan.Pricing {
				if idx == 0 {
					ret[reg.Slug][plan.Slug] = float64(pri.Price)
				}
				log.Printf("price: %v, taxed %v, currency: %s, unit: %s", pri.Price, pri.Taxed, pri.Currency, pri.Unit)
			}
		}
	}
	return ret, nil
}

func (cc *cherryClient) OSs() ([][2]string, error) {
	var plan string
	for k := range cherryPlans {
		plan = k // set plan to the first cherry plan, no matter what it is
		break
	}
	oss, _, err := cc.Images.List(plan, nil)
	if err != nil {
		return nil, err
	}
	ret := make([][2]string, len(oss))
	for i, os := range oss {
		ret[i] = [2]string{os.Slug, os.Name}
	}
	return ret, nil
}

func cherry() (client, error) {
	cc, err := cherrygo.NewClient()
	if err != nil {
		return nil, err
	}
	teams, _, err := cc.Teams.List(nil)
	if err != nil || len(teams) == 0 {
		return nil, err
	}
	return &cherryClient{
		teamID: teams[0].ID,
		Client: cc,
	}, nil
}
