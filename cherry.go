package main

import (
	"github.com/cherryservers/cherrygo/v3"
)

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

func (cc *cherryClient) Prices(dc string) (map[string]float64, error) {

	return nil, nil
}

func (cc *cherryClient) OSs() ([][2]string, error) {
	plans, err := cc.Machines()
	if err != nil {
		return nil, err
	}
	var ret [][2]string
	for idx, plan := range plans {
		oss, _, err := cc.Images.List(plan[0], nil)
		if err != nil {
			return nil, err
		}
		if idx == 0 {
			ret = make([][2]string, 0, len(plans)*len(oss))
		}
		for _, os := range oss {
			ret = append(ret, [2]string{
				os.Slug,
				plan[1] + ": " + os.Name,
			})
		}
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
