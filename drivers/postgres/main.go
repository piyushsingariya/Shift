package main

import (
	"github.com/gear5sh/gear5"
	"github.com/gear5sh/gear5/drivers/base"
	driver "github.com/gear5sh/gear5/drivers/postgres/internal"
	"github.com/gear5sh/gear5/protocol"
	_ "github.com/jackc/pgx/v4/stdlib"
)

func main() {
	driver := &driver.Postgres{
		Driver: base.NewBase(),
	}
	_ = protocol.BulkDriver(driver)

	defer driver.CloseConnection()
	gear5.RegisterDriver(driver)
}
