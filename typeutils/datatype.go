package typeutils

import (
	"fmt"
	"reflect"

	"github.com/gear5sh/gear5/types"
	"github.com/gear5sh/gear5/utils"
)

// TypeFromValue return DataType from v type
func TypeFromValue(v interface{}) types.DataType {
	switch reflect.TypeOf(v).Kind() {
	case reflect.Invalid:
		return types.NULL
	case reflect.Bool:
		return types.BOOL
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return types.INT64
	case reflect.Float32, reflect.Float64:
		return types.FLOAT64
	case reflect.String:
		// Check for specific formats
		if value, ok := v.(string); ok {
			// Check for email format
			if _, err := ReformatDate(value); err == nil {
				return types.TIMESTAMP
			}
		}

		return types.STRING
	case reflect.Slice, reflect.Array:
		return types.ARRAY
	case reflect.Map:
		return types.OBJECT
	default:
		return types.UNKNOWN
	}
}

func MaximumOnDataType[T any](typ types.DataType, a, b T) (T, error) {
	switch typ {
	case types.TIMESTAMP:
		adate, err := ReformatDate(a)
		if err != nil {
			return a, fmt.Errorf("failed to reformat[%v] while comparing: %s", a, err)
		}
		bdate, err := ReformatDate(b)
		if err != nil {
			return a, fmt.Errorf("failed to reformat[%v] while comparing: %s", b, err)
		}

		if utils.MaxDate(adate, bdate) == adate {
			return a, nil
		}

		return b, nil
	case types.INT64:
		aint, err := ReformatInt64(a)
		if err != nil {
			return a, fmt.Errorf("failed to reformat[%v] while comparing: %s", a, err)
		}

		bint, err := ReformatInt64(b)
		if err != nil {
			return a, fmt.Errorf("failed to reformat[%v] while comparing: %s", b, err)
		}

		if aint > bint {
			return a, nil
		}

		return b, nil
	default:
		return a, fmt.Errorf("comparison not available for data types %v now", typ)
	}
}
