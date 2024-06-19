package main

import (
	"github.com/gear5sh/gear5"
	driver "github.com/piyushsingariya/drivers/s3/internal"
)

func main() {
	driver := &driver.S3{}
	gear5.RegisterDriver(driver)
}
