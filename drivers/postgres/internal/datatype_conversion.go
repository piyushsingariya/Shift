package driver

import (
	"github.com/gear5sh/gear5/types"
)

var pgTypeToDataTypes = map[string]types.DataType{
	// integers
	"bigint":      types.INT64,
	"tinyint":     types.INT64,
	"integer":     types.INT64,
	"smallint":    types.INT64,
	"smallserial": types.INT64,
	"int":         types.INT64,
	"int2":        types.INT64,
	"int4":        types.INT64,
	"serial":      types.INT64,
	"serial2":     types.INT64,
	"serial4":     types.INT64,
	"serial8":     types.INT64,
	"bigserial":   types.INT64,

	// numbers
	"decimal":          types.FLOAT64,
	"numeric":          types.FLOAT64,
	"double precision": types.FLOAT64,
	"float":            types.FLOAT64,
	"float4":           types.FLOAT64,
	"float8":           types.FLOAT64,
	"real":             types.FLOAT64,

	// boolean
	"bool":    types.BOOL,
	"boolean": types.BOOL,

	// strings
	"bit varying":       types.STRING,
	"box":               types.STRING,
	"bytea":             types.STRING,
	"character":         types.STRING,
	"char":              types.STRING,
	"varbit":            types.STRING,
	"bit":               types.STRING,
	"bit(n)":            types.STRING,
	"varying(n)":        types.STRING,
	"cidr":              types.STRING,
	"inet":              types.STRING,
	"macaddr":           types.STRING,
	"macaddr8":          types.STRING,
	"character varying": types.STRING,
	"text":              types.STRING,
	"varchar":           types.STRING,
	"longvarchar":       types.STRING,
	"circle":            types.STRING,
	"hstore":            types.STRING,
	"name":              types.STRING,
	"uuid":              types.STRING,
	"json":              types.STRING,
	"jsonb":             types.STRING,
	"line":              types.STRING,
	"lseg":              types.STRING,
	"money":             types.STRING,
	"path":              types.STRING,
	"pg_lsn":            types.STRING,
	"point":             types.STRING,
	"polygon":           types.STRING,
	"tsquery":           types.STRING,
	"tsvector":          types.STRING,
	"xml":               types.STRING,
	"enum":              types.STRING,
	"tsrange":           types.STRING,

	// date/time
	"time":                        types.TIMESTAMP,
	"timez":                       types.TIMESTAMP,
	"date":                        types.TIMESTAMP,
	"timestamp":                   types.TIMESTAMP,
	"timestampz":                  types.TIMESTAMP,
	"interval":                    types.INT64,
	"timestamp with time zone":    types.TIMESTAMP,
	"timestamp without time zone": types.TIMESTAMP,

	// arrays
	"ARRAY": types.ARRAY,
	"array": types.ARRAY,
}
