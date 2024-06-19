package main

import (
	"github.com/gear5sh/gear5"
	driver "github.com/gear5sh/gear5/drivers/google-sheets/internal"
)

func main() {
	driver := &driver.GoogleSheets{}
	gear5.RegisterDriver(driver)
}
