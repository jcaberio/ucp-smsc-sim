package main

import (
	"github.com/jcaberio/ucp-smsc-sim/server"
	"github.com/jcaberio/ucp-smsc-sim/ui"
	"github.com/jcaberio/ucp-smsc-sim/util"
)

func main() {

	conf := util.Config{
		User:       "emi_client",
		Password:   "password",
		AccessCode: "2929",
		Port:       16004,
		HttpAddr:   ":16003",
		DNDelay:    2000,
		Tariff: map[string]float64{
			"01000001C1230001F0": 1,
			"01000001C123000250": 2,
			"01000001C123000210": 2.5,
			"01000001C123000220": 5,
			"01000001C123000230": 10,
			"01000001C123000240": 15,
		},
	}
	go ui.Render(conf)
	server.Start(conf)
}
