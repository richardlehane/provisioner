package main

import (
	"fmt"

	"github.com/cherryservers/cherrygo/v3"
)

var (
	cherryOS    = "ubuntu_20_04"
	cherryDC    = "eu_nord_1" // default region
	cherryPlans = stdPrices{
		"e3_1240v3":  0.188, // https://www.cherryservers.com/pricing/dedicated-servers/e3_1240v3?b=37&r=1 4 cores @ 3.4GHz, 16GB ECC DDR3 RAM, 2x SSD 250GB
		"e3_1240v5":  0.197, // https://www.cherryservers.com/pricing/dedicated-servers/e5_1620v4?b=37&r=1 4 cores @ 3.5GHz, 32GB ECC DDR4 RAM, 2x SSD 250GB
		"e3_1240lv5": 0.197, // https://www.cherryservers.com/pricing/dedicated-servers/e3_1240v5?b=37&r=1 4 cores @ 3.5GHz, 32GB ECC DDR4 RAM, 2x SSD 250GB
		"e5_1620v4":  0.197, // https://www.cherryservers.com/pricing/dedicated-servers/e3_1240lv5?b=37&r=1 4 cores @ 2.1GHz, 32GB ECC DDR4 RAM, 2x SSD 250GB
	}
)

type cherryClient struct {
	teamID    int
	projectID int
	*cherrygo.Client
}

func (cc *cherryClient) Provision(host, install, dc, plan string, price float64, spot bool) error {
	_, _, err := cc.Servers.Create(cc.provision(plan, host, dc, install, spot))
	return err
}

func (cc *cherryClient) Delete(host string) error {
	svrs, _, err := cc.Servers.List(cc.projectID, nil)
	if err != nil {
		return err
	}
	for _, svr := range svrs {
		if svr.Hostname == host {
			_, _, err = cc.Servers.Delete(svr.ID)
			return err
		}
	}
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
			if reg.SpotQty < 1 && reg.StockQty < 1 {
				continue
			}
			if _, ok := ret[reg.Slug]; !ok {
				ret[reg.Slug] = make(map[string]float64)
			}
		outer:
			for _, pri := range plan.Pricing {
				switch pri.Unit {
				case "Spot hourly":
					if reg.SpotQty > 0 {
						ret[reg.Slug][plan.Slug] = float64(pri.Price)
						break outer
					}
				case "Hourly":
					if reg.SpotQty < 1 {
						ret[reg.Slug][plan.Slug] = 0 - float64(pri.Price)
						break outer
					}
				default:
					continue outer
				}
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

func cherry(project string) (client, error) {
	cc, err := cherrygo.NewClient()
	if err != nil {
		return nil, err
	}
	teams, _, err := cc.Teams.List(nil)
	if err != nil || len(teams) == 0 {
		return nil, err
	}
	projs, _, err := cc.Projects.List(teams[0].ID, nil)
	if err != nil {
		return nil, err
	}
	for _, proj := range projs {
		if proj.Name == project {
			return &cherryClient{
				teamID:    teams[0].ID,
				projectID: proj.ID,
				Client:    cc,
			}, nil
		}
	}
	return nil, fmt.Errorf("can't find project %s", project)
}

func (cc *cherryClient) provision(plan, host, dc, install string, spot bool) *cherrygo.CreateServer {
	return &cherrygo.CreateServer{
		ProjectID:    cc.projectID,
		Plan:         plan, //plan slug
		Hostname:     host,
		Image:        cherryOS, //image slug
		Region:       dc,       // region slug
		UserData:     install,
		SpotInstance: spot,
	}
}
