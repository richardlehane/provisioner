package main

type joint struct {
	clients []client
}

func (j *joint) Provision(host, install string) error { return nil }
func (j *joint) Delete(host string) error             { return nil }
func (j *joint) Arbitrage(max float64) (string, string, float64, bool) {
	return "", "", 0, false
}

func (j *joint) SetDC(dc string) bool {
	found := -1
	for i, c := range j.clients {
		if c.SetDC(dc) {
			found = i
			break
		}
	}
	if found < 0 {
		return false
	}
	j.clients = []client{j.clients[found]}
	return true
}

func (j *joint) SetPlan(plan string) bool {
	found := -1
	for i, c := range j.clients {
		if c.SetPlan(plan) {
			found = i
			break
		}
	}
	if found < 0 {
		return false
	}
	j.clients = []client{j.clients[found]}
	return true
}

func (j *joint) SetSpot(spot bool) {
	for i := range j.clients {
		j.clients[i].SetSpot(spot)
	}
}

func (j *joint) Facilities() ([][2]string, error) {
	ret := make([][2]string, 0, 100)
	for _, c := range j.clients {
		fac, err := c.Facilities()
		if err != nil {
			return nil, err
		}
		for _, f := range fac {
			ret = append(ret, f)
		}
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
		for _, m := range mac {
			ret = append(ret, m)
		}
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
		for _, o := range os {
			ret = append(ret, o)
		}
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
