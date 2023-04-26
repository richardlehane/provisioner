package main

import (
	"testing"
)

var testClient client

func setup(t *testing.T) {
	var err error
	if testClient != nil {
		return
	}
	testClient, err = equinix("")
	if err != nil {
		t.Fatal(err)
	}
}

func TestFacilities(t *testing.T) {
	setup(t)
	fac, err := testClient.Facilities()
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range fac {
		t.Logf("code: %s; name %s\n", v[0], v[1])
	}
}

func TestPlans(t *testing.T) {
	setup(t)
	pla, err := testClient.Machines()
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range pla {
		t.Logf("slug: %s; name: %s\n",
			v[0], v[1])
	}
}

func TestPrices(t *testing.T) {
	setup(t)
	pri, err := testClient.Prices("sv15")
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range pri {
		t.Logf("%s price: %v\n", k, v)
	}
}

func TestOSes(t *testing.T) {
	setup(t)
	oss, err := testClient.OSs()
	if err != nil {
		t.Fatal(err)
	}
	for _, os := range oss {
		t.Logf("Slug: %s; Details: %s\n", os[0], os[1])
	}
}

func TestInstall(t *testing.T) {
	inst := readInstall("scripts/bench.yaml", "", "")
	t.Log(inst)
}
