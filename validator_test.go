package validator

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/x88/null"
)

func Test_Null(t *testing.T) {
	structEmpty := null.Int{}
	structNotEmpty := null.IntFrom(42)
	assert.NotNil(t, notEmpty(structEmpty, ``))
	assert.Nil(t, notEmpty(structNotEmpty, ``))
}

func TestValidator_Validate(t *testing.T) {
	testStruct := struct {
		Min   int    `validate:"min=3"`
		Max   int    `validate:"max=0"`
		Empty int    `validate:"empty=''"`
		In    int    `validate:"in='2,3,4,5'"`
		Type  string `validate:"type=base64"`

		CustomMsg   int `validate:"min=3,msg_min=msg1{param}msg2"`
		CustomAlias int `validate:"min=3,attr=custom_alias"`
	}{
		Min:   1,
		Max:   1,
		Empty: 0,
		In:    1,
		Type:  "test_string",

		CustomMsg:   1,
		CustomAlias: 1,
	}

	errs := Validate(testStruct)
	assert.Equal(t, false, errs.IsEmpty())

	assert.Equal(t, ErrMin, errs["Min"])
	assert.Equal(t, ErrMax, errs["Max"])
	assert.Equal(t, ErrZeroValue, errs["Empty"])
	assert.Equal(t, ErrInvalidValue, errs["In"])
	assert.Equal(t, ErrInvalidTypedValue, errs["Type"])
	assert.Equal(t, "msg13msg2", errs["CustomMsg"].Error())
	assert.Equal(t, ErrMin, errs["custom_alias"])
}

func TestValidator_ParseTags(t *testing.T) {
	data := []struct {
		str  string
		tags tagList
	}{
		{"name=value", []tag{{"name", "value"}}},
		{"under_score=value", []tag{{"under_score", "value"}}},
		{"name1=value1,name2=value2",
			[]tag{{"name1", "value1"}, {"name2", "value2"}}},
		{"quoted='value'", []tag{{"quoted", "value"}}},
		// value containing commas must be single-quoted
		{"quoted='v,a,l,u,e'", []tag{{"quoted", "v,a,l,u,e"}}},
		{"quoted='v,a,l,u,e',name1=val1",
			[]tag{{"quoted", "v,a,l,u,e"}, {"name1", "val1"}}},
	}
	validator := &Validator{}

	for _, cs := range data {
		tags, err := validator.parseTags(cs.str)
		assert.Nil(t, err, cs.str)

		assert.Equal(t, cs.tags, tags, cs.str)
	}
}

func TestIn(t *testing.T) {
	data := []struct {
		v     interface{}
		param string
		err   error
	}{
		// int
		{1, "2,3,4", ErrInvalidValue},
		{1, "1,2,3", nil},
		// float
		{1.1, "2.2,3,4", ErrInvalidValue},
		{1.1, "1.1,2.3,3", nil},
		// string
		{"str1", "str2,str3,str4", ErrInvalidValue},
		{"str1", "str1,str2,str3", nil},
		// invalid parameters
		{1.1, "2.2,3,4,not_float", ErrBadParameter},
	}

	for _, row := range data {
		err := in(row.v, row.param)
		assert.Equal(t, row.err, err, fmt.Sprintf("%#v", row))
	}
}

func TestTypeValid(t *testing.T) {
	data := []struct {
		v     interface{}
		param string
		err   error
	}{
		{"not_base64", "base64", ErrInvalidTypedValue},
		{"dGVzdA==", "base64", nil},

		{"not_timestamp", "timestamp", ErrInvalidTypedValue},
		{"2008-09-08T22:47:31-07:00", "timestamp", nil},
	}

	for _, row := range data {
		err := typeValid(row.v, row.param)
		assert.Equal(t, row.err, err, fmt.Sprintf("%#v", row))
	}
}

func Example() {
	testStruct := struct {
		Min int    `validate:"min=3"`
		In  string `validate:"attr=in_field,in='2,3,4,5',msg_in='not one of {param}'"`
	}{
		Min: 1,
		In:  "1",
	}

	errs := Validate(testStruct)
	if !errs.IsEmpty() {
		fmt.Println(errs["Min"])
		fmt.Println(errs["in_field"])
	}

	// Output: less than min
	// not one of 2,3,4,5
}
