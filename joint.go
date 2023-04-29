package main

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

func (j *joint) Provision(host, install, dc, plan string, price float64, spot bool) error { return nil }
func (j *joint) Delete(host string) error                                                 { return nil }
func (j *joint) Arbitrage(max float64) (string, string, float64, bool) {
	return "", "", 0, false
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
