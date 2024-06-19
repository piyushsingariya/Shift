package jdbc

import (
	"fmt"
	"strings"

	"github.com/gear5sh/gear5/protocol"
	"github.com/gear5sh/gear5/types"
)

const CDCDeletedAt = "_cdc_deleted_at"
const CDCLSN = "_cdc_lsn"
const CDCUpdatedAt = "_cdc_updated_at"

var CDCColumns = map[string]types.DataType{
	CDCDeletedAt: types.TIMESTAMP,
	CDCLSN:       types.STRING,
	CDCUpdatedAt: types.TIMESTAMP,
}

// Order by Cursor
func PostgresWithoutState(stream protocol.Stream) string {
	return fmt.Sprintf(`SELECT * FROM "%s"."%s" ORDER BY %s`, stream.Namespace(), stream.Name(), stream.Cursor())
}

// Order by Cursor
func PostgresWithState(stream protocol.Stream) string {
	return fmt.Sprintf(`SELECT * FROM "%s"."%s" where "%s">$1 ORDER BY "%s" ASC NULLS FIRST`, stream.Namespace(), stream.Name(), stream.Cursor(), stream.Cursor())
}

// Order by primary keys
func PostgresFullRefresh(stream protocol.Stream) string {
	return fmt.Sprintf(`SELECT * FROM "%s"."%s" ORDER BY %s`, stream.Namespace(), stream.Name(),
		strings.Join(stream.GetStream().SourceDefinedPrimaryKey.Array(), ", "))
}
