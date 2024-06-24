package main

import "fmt"

type joint struct {
	clients []client
}

func joinStd(a, b stdPrices) stdPrices {
	ret := make(stdPrices)
	for k, v := range a {
		ret[k] = v
	}
	for k, v := range b {
		ret[k] = v
	}
	return ret
}

func checkPlan(c client, p string) bool {
	plans, err := c.Machines()
	if err != nil {
		return false
	}
	for _, plan := range plans {
		if plan[0] == p {
			return true
		}
	}
	return false
}

func (j *joint) Provision(host, install, dc, plan string, price float32, spot bool) error {
	for _, c := range j.clients {
		if checkPlan(c, plan) {
			return c.Provision(host, install, dc, plan, price, spot)
		}
	}
	return fmt.Errorf("can't find plan %s", plan)
}

func (j *joint) Delete(host string) error { // return err if found, otherwise return the last not found error
	var err error
	for _, c := range j.clients {
		err = c.Delete(host)
		if err == nil {
			return nil
		}
	}
	return err
}

func (j *joint) Facilities() ([][2]string, error) {
	ret := make([][2]string, 0, 100)
	for _, c := range j.clients {
		fac, err := c.Facilities()
		if err != nil {
			return nil, err
		}
		ret = append(ret, fac...)
	}
	return ret, nil
}
func (j *joint) Machines() ([][2]string, error) {
	ret := make([][2]string, 0, 100)
	for _, c := range j.clients {
		mac, err := c.Machines()
		if err != nil {
			return nil, err
		}
		ret = append(ret, mac...)
	}
	return ret, nil
}
func (j *joint) OSs() ([][2]string, error) {
	ret := make([][2]string, 0, 100)
	for _, c := range j.clients {
		os, err := c.OSs()
		if err != nil {
			return nil, err
		}
		ret = append(ret, os...)
	}
	return ret, nil
}
func (j *joint) Prices() (dcMachinePrices, error) {
	ret := make(dcMachinePrices)
	for _, c := range j.clients {
		pri, err := c.Prices()
		if err != nil {
			return nil, err
		}
		for k, v := range pri {
			ret[k] = v
		}
	}
	return ret, nil
}
