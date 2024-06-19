package main

import (
	"github.com/gear5sh/gear5"
	driver "github.com/gear5sh/gear5/drivers/hubspot/internal"
)

func main() {
	driver := &driver.Hubspot{}
	gear5.RegisterDriver(driver)
}
