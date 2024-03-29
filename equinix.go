package main

import (
	"fmt"
	"log"
	"time"

	"github.com/packethost/packngo"
)

var (
	equinixOS    = "ubuntu_22_04"
	equinixDC    = "sv15"
	equinixPlans = stdPrices{
		"c3.medium.x86": 1.5,  // https://metal.equinix.com/product/servers/c3-medium/ 24 cores @ 2.8 GHz, 64GB DDR4 RAM, 960 GB SSD
		"m3.small.x86":  1.05, // name: m3.small.x86 https://metal.equinix.com/product/servers/m3-small/ 8 cores @ 2.8 GHz, 64GB RAM, 960 GB SSD
		"m3.large.x86":  3.1,  // https://metal.equinix.com/product/servers/m3-large/ 32 cores @ 2.5 GHz, 256GB DDR4 RAM, 2 x 3.8 TB NVMe
		"s3.xlarge.x86": 2.95, // https://metal.equinix.com/product/servers/s3-xlarge/ 24 cores @ 2.2 GHz, 192GB DDR4 RAM, 1.9 TB SSD
	}
)

type equinixClient struct {
	projectID  string
	facilities [][2]string
	machines   [][2]string
	prices     dcMachinePrices
	*packngo.Client
}

func (ec *equinixClient) Provision(host, install, dc, plan string, price float64, spot bool) error {
	var os string
	if *osf != "" {
		os = *osf
	} else {
		os = equinixOS
	}
	_, _, err := ec.Devices.Create(provision(ec.projectID, dc, plan, os, host, install, spot, price))
	return err
}

func (ec *equinixClient) Delete(host string) error {
	devices, _, err := ec.Devices.List(ec.projectID, nil)
	if err != nil {
		return err
	}
	var did string
	for _, d := range devices {
		if d.Hostname == host {
			did = d.ID
			break
		}
	}
	if did == "" {
		return fmt.Errorf("host not found: %s", host)
	}
	if *dryf {
		log.Printf("dry run: deleting %s from equinix", host)
		return nil
	}
	_, err = ec.Devices.Delete(did, true)
	return err
}

func (ec *equinixClient) Facilities() ([][2]string, error) {
	if ec.facilities != nil {
		return ec.facilities, nil
	}
	fac, _, err := ec.Client.Facilities.List(nil)
	if err != nil {
		return nil, err
	}
	ret := make([][2]string, len(fac))
	for idx, v := range fac {
		ret[idx][0], ret[idx][1] = v.Code, v.Name
	}
	ec.facilities = ret
	return ret, nil
}

func (ec *equinixClient) Machines() ([][2]string, error) {
	if ec.machines != nil {
		return ec.machines, nil
	}
	pla, _, err := ec.Plans.List(nil)
	if err != nil {
		return nil, err
	}
	ret := make([][2]string, len(pla))
	for idx, v := range pla {
		ret[idx][0], ret[idx][1] = v.Slug, v.Name
	}
	ec.machines = ret
	return ret, nil
}

func (ec *equinixClient) Prices() (dcMachinePrices, error) {
	if ec.prices != nil {
		return ec.prices, nil
	}
	pri, _, err := ec.SpotMarket.PricesByFacility()
	if err != nil {
		return nil, err
	}
	ret := filterPlans(dcMachinePrices(pri), listPlans(equinixPlans))
	ec.prices = ret
	return ret, nil
}

func (ec *equinixClient) OSs() ([][2]string, error) {
	oss, _, err := ec.OperatingSystems.List()
	if err != nil {
		return nil, err
	}
	ret := make([][2]string, len(oss))
	for idx, os := range oss {
		ret[idx][0], ret[idx][1] = os.Slug, fmt.Sprintf("%s / %s / %s", os.Distro, os.Name, os.Version)
	}
	return ret, nil
}

func equinix(project string) (client, error) {
	c, err := packngo.NewClient()
	if err != nil {
		log.Fatal(err)
	}
	var pid string
	if project != "" {
		ps, _, err := c.Projects.List(nil)
		if err != nil {
			log.Fatal(err)
		}
		for _, p := range ps {
			if project == p.Name {
				pid = p.ID
				break
			}
		}
		if pid == "" {
			return nil, fmt.Errorf("can't find project name %s", project)
		}
	}
	return &equinixClient{
		projectID: pid,
		Client:    c,
	}, nil
}

func provision(pid, dc, plan, os, host, install string, spot bool, pri float64) *packngo.DeviceCreateRequest {
	term := &packngo.Timestamp{Time: time.Now().Add(*lifef)}
	return &packngo.DeviceCreateRequest{
		Hostname:        host,
		Facility:        []string{dc},
		Plan:            plan,
		OS:              os,
		ProjectID:       pid,
		UserData:        install,
		BillingCycle:    "hourly",
		SpotInstance:    spot,
		SpotPriceMax:    pri,
		TerminationTime: term,
	}
}
