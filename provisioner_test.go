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

func TestPrices(t *testing.T) {
	setup(t)
	pri, _, err := testClient.SpotMarket.Prices()
	if err != nil {
		t.Fatal(err)
	}
	sjc1 := pri["sjc1"]
	for k, v := range sjc1 {
		t.Logf("Machine: %v, Price: %v\n", k, v)
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
	inst := readInstall("scripts/bench.yaml")
	t.Logf(inst)
}
