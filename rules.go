package validator

import (
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	regexpBase64 = regexp.MustCompile("^(?:[A-Za-z0-9+\\/]{4})*(?:[A-Za-z0-9+\\/]{2}==|[A-Za-z0-9+\\/]{3}=|[A-Za-z0-9+\\/]{4})$")
)

// notempty tests whether a variable value non-zero
// as defined by the golang spec.
func notZero(v interface{}, param string) error {
	st := reflect.ValueOf(v)
	valid := true
	switch st.Kind() {
	case reflect.String:
		valid = st.String() != ``
	case reflect.Ptr, reflect.Interface:
		valid = !st.IsNil()
	case reflect.Slice, reflect.Map, reflect.Array:
		valid = st.Len() != 0
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		valid = st.Int() != 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		valid = st.Uint() != 0
	case reflect.Float32, reflect.Float64:
		valid = st.Float() != 0
	case reflect.Bool:
		valid = st.Bool()
	case reflect.Struct:
		interfaceType := reflect.TypeOf(v)
		if strings.Contains(strings.ToLower(interfaceType.String()), `null`) {
			if _, exists := interfaceType.FieldByName(`Valid`); exists {
				v := reflect.ValueOf(v)
				if v.FieldByName(`Valid`).Bool() {
					switch interfaceType.String() {
					case `sql.NullInt64`, `null.Int`:
						if _, exists = interfaceType.FieldByName(`Int64`); exists {
							valid = v.FieldByName(`Int64`).Int() != 0
						} else {
							valid = false
						}
					case `sql.NullString`, `null.String`:
						if _, exists = interfaceType.FieldByName(`String`); exists {
							valid = v.FieldByName(`String`).String() != ``
						} else {
							valid = false
						}
					case `sql.NullFloat64`, `null.Float`:
						if _, exists = interfaceType.FieldByName(`Float64`); exists {
							valid = v.FieldByName(`Float64`).Float() != 0
						} else {
							valid = false
						}
					default:
						return ErrUnsupported
					}
				} else {
					valid = false
				}
			} else {
				valid = false
			}
		} else {
			return ErrUnsupported
		}
	case reflect.Invalid:
		valid = false
	default:
		return ErrUnsupported
	}

	if !valid {
		return ErrZeroValue
	}
	return nil
}

func notEmpty(v interface{}, param string) error {
	st := reflect.ValueOf(v)
	valid := true
	switch st.Kind() {
	case reflect.String:
		valid = st.String() != ``
	case reflect.Ptr, reflect.Interface:
		valid = !st.IsNil()
	case reflect.Slice, reflect.Map, reflect.Array:
		valid = st.Len() != 0
	case reflect.Struct:
		interfaceType := reflect.TypeOf(v)
		if strings.Contains(strings.ToLower(interfaceType.String()), `null`) {
			if _, exists := interfaceType.FieldByName(`Valid`); exists {
				valid = reflect.ValueOf(v).FieldByName(`Valid`).Bool()
			}
		} else {
			return ErrUnsupported
		}
	case reflect.Invalid:
		valid = false
	default:
		return ErrUnsupported
	}

	if !valid {
		return ErrZeroValue
	}
	return nil
}

// length tests whether a variable's length is equal to a given
// value. For strings it tests the number of characters whereas
// for maps and slices it tests the number of items.
func length(v interface{}, param string) error {
	st := reflect.ValueOf(v)
	valid := true
	switch st.Kind() {
	case reflect.String:
		p, err := asInt(param)
		if err != nil {
			return ErrBadParameter
		}
		valid = int64(len(st.String())) == p
	case reflect.Slice, reflect.Map, reflect.Array:
		p, err := asInt(param)
		if err != nil {
			return ErrBadParameter
		}
		valid = int64(st.Len()) == p
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		p, err := asInt(param)
		if err != nil {
			return ErrBadParameter
		}
		valid = st.Int() == p
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		p, err := asUint(param)
		if err != nil {
			return ErrBadParameter
		}
		valid = st.Uint() == p
	case reflect.Float32, reflect.Float64:
		p, err := asFloat(param)
		if err != nil {
			return ErrBadParameter
		}
		valid = st.Float() == p
	default:
		return ErrUnsupported
	}
	if !valid {
		return ErrLen
	}
	return nil
}

// min tests whether a variable value is larger or equal to a given
// number. For number types, it's a simple lesser-than test; for
// strings it tests the number of characters whereas for maps
// and slices it tests the number of items.
func min(v interface{}, param string) error {
	st := reflect.ValueOf(v)
	invalid := false
	switch st.Kind() {
	case reflect.String:
		p, err := asInt(param)
		if err != nil {
			return ErrBadParameter
		}
		invalid = int64(len(st.String())) < p
	case reflect.Slice, reflect.Map, reflect.Array:
		p, err := asInt(param)
		if err != nil {
			return ErrBadParameter
		}
		invalid = int64(st.Len()) < p
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		p, err := asInt(param)
		if err != nil {
			return ErrBadParameter
		}
		invalid = st.Int() < p
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		p, err := asUint(param)
		if err != nil {
			return ErrBadParameter
		}
		invalid = st.Uint() < p
	case reflect.Float32, reflect.Float64:
		p, err := asFloat(param)
		if err != nil {
			return ErrBadParameter
		}
		invalid = st.Float() < p
	default:
		return ErrUnsupported
	}
	if invalid {
		return ErrMin
	}
	return nil
}

// max tests whether a variable value is lesser than a given
// value. For numbers, it's a simple lesser-than test; for
// strings it tests the number of characters whereas for maps
// and slices it tests the number of items.
func max(v interface{}, param string) error {
	st := reflect.ValueOf(v)
	var invalid bool
	switch st.Kind() {
	case reflect.String:
		p, err := asInt(param)
		if err != nil {
			return ErrBadParameter
		}
		invalid = int64(len(st.String())) > p
	case reflect.Slice, reflect.Map, reflect.Array:
		p, err := asInt(param)
		if err != nil {
			return ErrBadParameter
		}
		invalid = int64(st.Len()) > p
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		p, err := asInt(param)
		if err != nil {
			return ErrBadParameter
		}
		invalid = st.Int() > p
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		p, err := asUint(param)
		if err != nil {
			return ErrBadParameter
		}
		invalid = st.Uint() > p
	case reflect.Float32, reflect.Float64:
		p, err := asFloat(param)
		if err != nil {
			return ErrBadParameter
		}
		invalid = st.Float() > p
	default:
		return ErrUnsupported
	}
	if invalid {
		return ErrMax
	}
	return nil
}

// regex is the builtin validation function that checks
// whether the string variable matches a regular expression
func regex(v interface{}, param string) error {
	s, ok := v.(string)
	if !ok {
		return ErrUnsupported
	}

	re, err := regexp.Compile(param)
	if err != nil {
		return ErrBadParameter
	}

	if !re.MatchString(s) {
		return ErrRegexp
	}
	return nil
}

// in is the builtin validation function that checks
// whether the value is listed in the list of supported values.
// Works with: int, uint, float, string
func in(v interface{}, param string) error {
	var (
		st               = reflect.ValueOf(v)
		params           = strings.Split(param, ",")
		expectedValues   = make([]interface{}, 0)
		actualValueTyped interface{}
	)

	switch st.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		for _, p := range params {
			vInt, err := asInt(p)
			if err != nil {
				return ErrBadParameter
			}

			expectedValues = append(expectedValues, vInt)
		}

		actualValueTyped = st.Int()
	case reflect.Float32, reflect.Float64:
		for _, p := range params {
			vFloat, err := asFloat(p)
			if err != nil {
				return ErrBadParameter
			}

			expectedValues = append(expectedValues, vFloat)
		}

		actualValueTyped = st.Float()
	case reflect.String:
		for _, p := range params {
			expectedValues = append(expectedValues, p)
		}

		actualValueTyped = st.String()
	default:
		return ErrBadParameter
	}

	var (
		expectedValuesR = reflect.ValueOf(expectedValues)
		found           bool
	)

	for j := 0; j < expectedValuesR.Len(); j++ {
		matchElem := expectedValuesR.Index(j).Interface()
		if reflect.DeepEqual(reflect.ValueOf(actualValueTyped).Interface(), matchElem) {
			found = true
			break
		}
	}

	if !found {
		return ErrInvalidValue
	}

	return nil
}

// typeValid is the builtin validation function that checks
// if the value is valid for provided type
// Supported types: timestamp, base64
func typeValid(v interface{}, param string) error {
	str := reflect.ValueOf(v).String()

	switch param {
	case "timestamp":
		_, err := time.Parse(time.RFC3339, str)
		if err != nil {
			return ErrInvalidTypedValue
		}
	case "base64":
		if !regexpBase64.MatchString(str) {
			return ErrInvalidTypedValue
		}
	default:
		return ErrBadParameter
	}

	return nil
}

// asInt retuns the parameter as a int64
// or panics if it can't convert
func asInt(param string) (int64, error) {
	i, err := strconv.ParseInt(param, 0, 64)
	if err != nil {
		return 0, ErrBadParameter
	}
	return i, nil
}

// asUint retuns the parameter as a uint64
// or panics if it can't convert
func asUint(param string) (uint64, error) {
	i, err := strconv.ParseUint(param, 0, 64)
	if err != nil {
		return 0, ErrBadParameter
	}
	return i, nil
}

// asFloat retuns the parameter as a float64
// or panics if it can't convert
func asFloat(param string) (float64, error) {
	i, err := strconv.ParseFloat(param, 64)
	if err != nil {
		return 0.0, ErrBadParameter
	}
	return i, nil
}
