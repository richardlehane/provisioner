package main

import (
	"testing"
)

var testClient client
var thisService service

type service int

const (
	JOINT service = iota
	METAL
	CHERRY
)

func setup(t *testing.T, s service) {
	var err error
	if testClient != nil && thisService == s {
		return
	}
	thisService = s
	switch s {
	case JOINT:
		cClient, err := cherry("bench")
		if err != nil {
			t.Fatal(err)
		}
		eClient, err := equinix("bench")
		if err != nil {
			t.Fatal(err)
		}
		testClient = &joint{[]client{cClient, eClient}}
	case METAL:
		testClient, err = equinix("bench")
		if err != nil {
			t.Fatal(err)
		}
	case CHERRY:
		testClient, err = cherry("bench")
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestFacilities(t *testing.T) {
	setup(t, JOINT)
	fac, err := testClient.Facilities()
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range fac {
		t.Logf("code: %s; name %s\n", v[0], v[1])
	}
}

func TestPlans(t *testing.T) {
	setup(t, JOINT)
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
	setup(t, JOINT)
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
	setup(t, JOINT)
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
