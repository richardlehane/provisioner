package main

import (
	"testing"

	"github.com/packethost/packngo"
)

var testClient *packngo.Client

func setup(t *testing.T) {
	var err error
	if testClient != nil {
		return
	}
	testClient, err = packngo.NewClient()
	if err != nil {
		t.Fatal(err)
	}
	return
}

func TestFacilities(t *testing.T) {
	setup(t)
	fac, _, err := testClient.Facilities.List(nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range fac {
		t.Logf("code: %s; name %s\n", v.Code, v.Name)
	}
}

func TestPlans(t *testing.T) {
	setup(t)
	pla, _, err := testClient.Plans.List(nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range pla {
		t.Logf("slug: %s; name: %s\n",
			v.Slug, v.Name)
	}
}

func TestPrices(t *testing.T) {
	setup(t)
	pri, _, err := testClient.SpotMarket.Prices()
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range pri["sjc1"] {
		t.Logf("%s price: %v\n", k, v)
	}
}

func TestOSes(t *testing.T) {
	setup(t)
	oss, _, err := testClient.OperatingSystems.List()
	if err != nil {
		t.Fatal(err)
	}
	for _, os := range oss {
		t.Logf("Distro: %s; Name: %s; Slug: %s; Version: %s\n", os.Distro, os.Name, os.Slug, os.Version)
	}
}

func TestInstall(t *testing.T) {
	inst := readInstall("scripts/bench.yaml", "", "")
	t.Log(inst)
}

func TestThrottle(t *testing.T) {
	err := throttle("https://www.itforarchivists.com/siegfried/develop", "div:nth-of-type(7) a", "365")
	t.Log(err)
}
