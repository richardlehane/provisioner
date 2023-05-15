package main

import "testing"

var testClient client

func setup(t *testing.T) {
	var err error
	if testClient != nil {
		return
	}
	cClient, err := cherry("bench")
	if err != nil {
		t.Fatal(err)
	}
	eClient, err := equinix("bench")
	if err != nil {
		t.Fatal(err)
	}
	testClient = &joint{[]client{cClient, eClient}}
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
	pri, err := testClient.Prices()
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range pri {
		for kk, vv := range v {
			t.Logf("%s %s price: %v\n", k, kk, vv)
		}
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

func TestSanitize(t *testing.T) {
	*replf = "CHERRY_AUTH_TOKEN,fakedata"
	inst := readInstall("scripts/bench.yaml", "", "")
	t.Log(sanitize(inst))
}
