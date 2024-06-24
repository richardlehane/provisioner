package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	metal "github.com/equinix/equinix-sdk-go/services/metalv1"
)

var (
	equinixOS    = "ubuntu_24_04"
	equinixMetro = "sv"
	equinixPlans = stdPrices{
		"c3.small.x86":  0.75, // https://deploy.equinix.com/product/servers/c3-small/ 8 cores @ 3.40 GHz, 32GB RAM, 960 GB SSD
		"c2.medium.x86": 1,    //24 cores @ 2.00GHz, 64 GB, 960 GB SSD
		"c3.medium.x86": 1.5,  // https://deploy.equinix.com/product/servers/c3-medium/ 24 cores @ 2.8 GHz, 64GB DDR4 RAM, 960 GB SSD
		"m3.small.x86":  1.05, // https://deploy.equinix.com/product/servers/m3-small/ 8 cores @ 2.8 GHz, 64GB RAM, 960 GB SSD
		"m3.large.x86":  3.1,  // https://deploy.equinix.com/product/servers/m3-large/ 32 cores @ 2.5 GHz, 256GB DDR4 RAM, 2 x 3.8 TB NVMe
		"s3.xlarge.x86": 2.95, // https://deploy.equinix.com/product/servers/s3-xlarge/ 24 cores @ 2.2 GHz, 192GB DDR4 RAM, 1.9 TB SSD
	}
)

type equinixClient struct {
	projectID string
	metros    [][2]string
	machines  [][2]string
	prices    dcMachinePrices
	*metal.APIClient
}

func (ec *equinixClient) Provision(host, install, dc, plan string, price float32, spot bool) error {
	var os string
	if *osf != "" {
		os = *osf
	} else {
		os = equinixOS
	}
	_, _, err := ec.DevicesApi.CreateDevice(context.Background(), ec.projectID).CreateDeviceRequest(provision(dc, plan, os, host, install, spot, price)).Execute()
	return err
}

func (ec *equinixClient) Delete(host string) error {
	devices, _, err := ec.DevicesApi.FindProjectDevices(context.Background(), ec.projectID).Execute()
	if err != nil {
		return err
	}
	var did string
	for _, d := range devices.Devices {
		if *d.Hostname == host {
			did = *d.Id
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
	_, err = ec.DevicesApi.DeleteDevice(context.Background(), did).Execute()
	return err
}

// Facilities really returns Metros as this is now the Equinix preference
func (ec *equinixClient) Facilities() ([][2]string, error) {
	if ec.metros != nil {
		return ec.metros, nil
	}
	met, _, err := ec.MetrosApi.FindMetros(context.Background()).Execute()
	if err != nil {
		return nil, err
	}
	ret := make([][2]string, len(met.Metros))
	for idx, v := range met.Metros {
		ret[idx][0], ret[idx][1] = *v.Code, *v.Name
	}
	ec.metros = ret
	return ret, nil
}

func (ec *equinixClient) Machines() ([][2]string, error) {
	if ec.machines != nil {
		return ec.machines, nil
	}
	pla, _, err := ec.PlansApi.FindPlans(context.Background()).Categories(
		[]metal.FindOrganizationDevicesCategoriesParameterInner{metal.FINDORGANIZATIONDEVICESCATEGORIESPARAMETERINNER_COMPUTE},
	).Type_(
		metal.FINDPLANSTYPEPARAMETER_STANDARD,
	).Execute()
	if err != nil {
		return nil, err
	}
	ret := make([][2]string, len(pla.Plans))
	for idx, v := range pla.Plans {
		ret[idx][0], ret[idx][1] = *v.Slug, *v.Name
	}
	ec.machines = ret
	return ret, nil
}

func (ec *equinixClient) standardPrices() (dcMachinePrices, error) {
	metromap := make(map[string]string)
	met, _, err := ec.MetrosApi.FindMetros(context.Background()).Execute()
	if err != nil {
		return nil, err
	}
	for _, m := range met.Metros {
		metromap[*m.Id] = *m.Code
	}
	pla, _, err := ec.PlansApi.FindPlans(context.Background()).Categories(
		[]metal.FindOrganizationDevicesCategoriesParameterInner{metal.FINDORGANIZATIONDEVICESCATEGORIESPARAMETERINNER_COMPUTE},
	).Type_(
		metal.FINDPLANSTYPEPARAMETER_STANDARD,
	).Execute()
	if err != nil {
		return nil, err
	}
	dcmp := make(dcMachinePrices)
	for _, v := range pla.Plans {
		slug := *v.Slug
		for _, vv := range v.AvailableInMetros {
			pri := vv.GetPrice()
			hr := pri.GetHour()
			if hr == 0 {
				continue
			}
			hash := strings.Split(*vv.Href, "/")
			metro := metromap[hash[len(hash)-1]]
			if _, ok := dcmp[metro]; !ok {
				dcmp[metro] = make(map[string]float32)
			}
			dcmp[metro][slug] = 0 - float32(hr)
		}
	}
	return dcmp, nil
}

func (ec *equinixClient) Prices() (dcMachinePrices, error) {
	if ec.prices != nil {
		return ec.prices, nil
	}
	pri, _, err := ec.SpotMarketApi.FindMetroSpotMarketPrices(context.Background()).Execute()
	if err != nil {
		return nil, err
	}
	m, err := pri.SpotMarketPrices.ToMap()
	dcmpri := make(dcMachinePrices)
	if err != nil {
		return nil, err
	}
	for k, v := range m {
		switch sppf := v.(type) {
		case *metal.SpotPricesPerFacility:
			dcmpri[k] = make(map[string]float32)
			mm, err := sppf.ToMap()
			if err != nil {
				return nil, err
			}
			for kk, vv := range mm {
				switch sppb := vv.(type) {
				case *metal.SpotPricesPerBaremetal:
					dcmpri[k][kk] = *sppb.Price
				}
			}
		}
	}
	pri2, err := ec.standardPrices()
	if err != nil {
		return nil, err
	}
	for k, v := range pri2 {
		if _, ok := dcmpri[k]; !ok {
			dcmpri[k] = v
		} else {
			for kk, vv := range v {
				if _, okok := dcmpri[k][kk]; !okok {
					dcmpri[k][kk] = vv
				}
			}
		}
	}
	ec.prices = dcmpri //filterPlans(dcmpri, listPlans(equinixPlans))
	return ec.prices, nil
}

func (ec *equinixClient) OSs() ([][2]string, error) {
	oss, _, err := ec.OperatingSystemsApi.FindOperatingSystems(context.Background()).Execute()
	if err != nil {
		return nil, err
	}
	ret := make([][2]string, len(oss.OperatingSystems))
	for idx, os := range oss.OperatingSystems {
		ret[idx][0], ret[idx][1] = *os.Slug, fmt.Sprintf("%s / %s / %s", *os.Distro, *os.Name, *os.Version)
	}
	return ret, nil
}

func equinix(project string) (client, error) {
	configuration := metal.NewConfiguration()
	configuration.AddDefaultHeader("X-Auth-Token", os.Getenv("PACKET_AUTH_TOKEN"))
	api_client := metal.NewAPIClient(configuration)
	var pid string
	if project != "" {
		ps, _, err := api_client.ProjectsApi.FindProjects(context.Background()).Execute()
		if err != nil {
			log.Fatal(err)
		}
		for _, p := range ps.Projects {
			if project == *p.Name {
				pid = *p.Id
				break
			}
		}
		if pid == "" {
			return nil, fmt.Errorf("can't find project name %s", project)
		}
	}
	return &equinixClient{
		projectID: pid,
		APIClient: api_client,
	}, nil
}

func provision(dc, plan, os, host, install string, spot bool, pri float32) metal.CreateDeviceRequest {
	tt := time.Now().Add(*lifef)
	return metal.CreateDeviceRequest{
		DeviceCreateInMetroInput: &metal.DeviceCreateInMetroInput{
			Metro:           dc,
			Hostname:        &host,
			Plan:            plan,
			OperatingSystem: os,
			Userdata:        &install,
			BillingCycle:    metal.DEVICECREATEINPUTBILLINGCYCLE_HOURLY.Ptr(),
			SpotInstance:    &spot,
			SpotPriceMax:    &pri,
			TerminationTime: &tt,
		},
	}
}
