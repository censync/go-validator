package validator

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"unicode"
)

// TextErr is an error that also implements the TextMarshaller interface for
// serializing out to various plain text encodings. Packages creating their
// own custom errors should use TextErr if they're intending to use serializing
// formats like json, msgpack etc.
type TextErr struct {
	Err error
}

// Error implements the error interface.
func (t TextErr) Error() string {
	return t.Err.Error()
}

// MarshalText implements the TextMarshaller
func (t TextErr) MarshalText() ([]byte, error) {
	return []byte(t.Err.Error()), nil
}

var (
	// ErrZeroValue is the error returned when variable has zero valud
	// and nonzero was specified
	ErrZeroValue = TextErr{errors.New("zero value")}
	// ErrMin is the error returned when variable is less than mininum
	// value specified
	ErrMin = TextErr{errors.New("less than min")}
	// ErrMax is the error returned when variable is more than
	// maximum specified
	ErrMax = TextErr{errors.New("greater than max")}
	// ErrLen is the error returned when length is not equal to
	// param specified
	ErrLen = TextErr{errors.New("invalid length")}
	// ErrRegexp is the error returned when the value does not
	// match the provided regular expression parameter
	ErrRegexp = TextErr{errors.New("regular expression mismatch")}
	// ErrUnsupported is the error error returned when a validation rule
	// is used with an unsupported variable type
	ErrUnsupported = TextErr{errors.New("unsupported type")}
	// ErrBadParameter is the error returned when an invalid parameter
	// is provided to a validation rule (e.g. a string where an int was
	// expected (max=foo,len=bar) or missing a parameter when one is required (len=))
	ErrBadParameter = TextErr{errors.New("bad parameter")}
	// ErrUnknownTag is the error returned when an unknown tag is found
	ErrUnknownTag = TextErr{errors.New("unknown tag")}
	// ErrInvalid is the error returned when variable is invalid
	// (normally a nil pointer)
	ErrInvalid = TextErr{errors.New("invalid value")}
	// ErrInvalidValue is the error error returned when a passed value
	// was not found in rule's list
	ErrInvalidValue = TextErr{errors.New("invalid value")}
	// ErrInvalidTypedValue is the error error returned when a passed value
	// doesn't correspond with defined type
	ErrInvalidTypedValue = TextErr{errors.New("invalid value for provided type")}

	// tagRegexp is a regexp for tags extraction
	tagRegexp = regexp.MustCompile("([^'=]+)=(?:'?)([^'=]*)(?:'?)(?:,|$)")
)

const (
	tagAttr = "attr"
)

// ErrorMap is a map which contains all errors from validating a struct.
type ErrorMap map[string]error

// ErrorMap implements the Error interface so we can check error against nil.
// The returned error is if existent the first error which was added to the map.
func (err ErrorMap) String() string {
	for k, err := range err {
		if err != nil {
			return fmt.Sprintf("%s: %s", k, err.Error())
		}
	}

	return ""
}


func (err ErrorMap) Error() error {
	for k, err := range err {
		if err != nil {
			return fmt.Errorf("%s: %s", k, err.Error())
		}
	}

	return nil
}

// IsEmpty returns true if the map consists no errors
func (err ErrorMap) IsEmpty() bool {
	return len(err) == 0
}

// ErrorArray is a slice of errors returned by the Validate function.
type ErrorArray []error

// ErrorArray implements the Error interface and returns the first error as
// string if existent.
func (err ErrorArray) Error() string {
	if len(err) > 0 {
		return err[0].Error()
	}
	return ""
}

// ValidationFunc is a function that receives the value of a
// field and a parameter used for the respective validation tag.
type ValidationFunc func(v interface{}, param string) error

// Validator implements a validator
type Validator struct {
	// Tag name being used.
	tagName string
	// validationFuncs is a map of ValidationFuncs indexed
	// by their name.
	validationFuncs map[string]ValidationFunc
}

// Helper validator so users can use the
// functions directly from the package
var defaultValidator = NewValidator()

// NewValidator creates a new Validator
func NewValidator() *Validator {
	return &Validator{
		tagName: "validate",
		validationFuncs: map[string]ValidationFunc{
			"notempty": notZero,
			"empty":    notZero,
			"len":      length,
			"min":      min,
			"max":      max,
			"regexp":   regex,
			"in":       in,
			"type":     typeValid,
		},
	}
}

// SetTag allows you to change the tag name used in structs
func SetTag(tag string) {
	defaultValidator.SetTag(tag)
}

// SetTag allows you to change the tag name used in structs
func (mv *Validator) SetTag(tag string) {
	mv.tagName = tag
}

// WithTag creates a new Validator with the new tag name. It is
// useful to chain-call with Validate so we don't change the tag
// name permanently: validator.WithTag("foo").Validate(t)
func WithTag(tag string) *Validator {
	return defaultValidator.WithTag(tag)
}

// WithTag creates a new Validator with the new tag name. It is
// useful to chain-call with Validate so we don't change the tag
// name permanently: validator.WithTag("foo").Validate(t)
func (mv *Validator) WithTag(tag string) *Validator {
	v := mv.copy()
	v.SetTag(tag)
	return v
}

// Copy a validator
func (mv *Validator) copy() *Validator {
	return &Validator{
		tagName:         mv.tagName,
		validationFuncs: mv.validationFuncs,
	}
}

// SetValidationFunc sets the function to be used for a given
// validation constraint. Calling this function with nil vf
// is the same as removing the constraint function from the list.
func SetValidationFunc(name string, vf ValidationFunc) error {
	return defaultValidator.SetValidationFunc(name, vf)
}

// SetValidationFunc sets the function to be used for a given
// validation constraint. Calling this function with nil vf
// is the same as removing the constraint function from the list.
func (mv *Validator) SetValidationFunc(name string, vf ValidationFunc) error {
	if name == "" {
		return errors.New("name cannot be empty")
	}
	if vf == nil {
		delete(mv.validationFuncs, name)
		return nil
	}
	mv.validationFuncs[name] = vf
	return nil
}

// Validate validates the fields of a struct based
// on 'validator' tags and returns errors found indexed
// by the field name.
func Validate(v interface{}) ErrorMap {
	return defaultValidator.Validate(v)
}

// Validate validates the fields of a struct based
// on 'validator' tags and returns errors found indexed
// by the field name.
func (mv *Validator) Validate(v interface{}) ErrorMap {
	var (
		sv = reflect.ValueOf(v)
		st = reflect.TypeOf(v)
		m  = make(ErrorMap)
	)

	if sv.Kind() == reflect.Ptr && !sv.IsNil() {
		return mv.Validate(sv.Elem().Interface())
	}
	if sv.Kind() != reflect.Struct {
		m["_summary"] = ErrUnsupported
		return m
	}

	nfields := sv.NumField()
	for i := 0; i < nfields; i++ {
		var (
			f     = sv.Field(i)
			fname = st.Field(i).Name
			errs  ErrorArray
		)

		// deal with pointers
		for f.Kind() == reflect.Ptr && !f.IsNil() {
			f = f.Elem()
		}

		tag := st.Field(i).Tag.Get(mv.tagName)
		if tag == "-" || (tag == "" && f.Kind() != reflect.Struct) {
			continue
		}

		// parse tags on the highest level to pass further
		tags, err := mv.parseTags(tag)
		if err != nil {
			m[fname] = err
			continue
		}

		// custom field alias
		if nameTag, exists := tags.getByName(tagAttr); exists {
			fname = nameTag.Param
		}

		switch f.Kind() {
		// nested struct
		case reflect.Struct:
			if !unicode.IsUpper(rune(fname[0])) {
				continue
			}

			e := mv.Validate(f.Interface())
			for j, k := range e {
				// Nested struct gets alias of parent struct
				// as a prefix
				m[fname+"."+j] = k
			}

			// flat struct
		default:
			err := mv.valid(f.Interface(), tags)
			if errors, ok := err.(ErrorArray); ok {
				errs = errors
			} else {
				if err != nil {
					errs = ErrorArray{err}
				}
			}
		}

		if len(errs) > 0 {
			m[fname] = errs[0]
		}
	}

	return m
}

// Valid validates a value based on the provided
// tags and returns errors found or nil.
func Valid(val interface{}, tags string) error {
	return defaultValidator.Valid(val, tags)
}

// Valid validates a value based on the *raw string*
// tags and returns errors found or nil.
func (mv *Validator) Valid(val interface{}, tagsRaw string) error {
	if tagsRaw == "-" {
		return nil
	}

	tags, err := mv.parseTags(tagsRaw)
	if err != nil {
		// unknown tag found, give up.
		return err
	}

	return mv.valid(val, tags)
}

// Valid validates a value based on the provided
// tags and returns errors found or nil.
func (mv *Validator) valid(val interface{}, tags tagList) error {
	v := reflect.ValueOf(val)
	if v.Kind() == reflect.Ptr && !v.IsNil() {
		return mv.valid(v.Elem().Interface(), tags)
	}

	var err error
	switch v.Kind() {
	case reflect.Struct:
		return ErrUnsupported
	case reflect.Invalid:
		err = mv.validateVar(nil, tags)
	default:
		err = mv.validateVar(val, tags)
	}

	return err
}

// validateVar validates one single variable
func (mv *Validator) validateVar(v interface{}, tags tagList) error {
	errs := make(ErrorArray, 0, len(tags))
	for _, t := range tags {
		fn, found := mv.validationFuncs[t.Name]
		if !found {
			// skip additional tags
			if strings.HasPrefix(t.Name, "msg_") || t.Name == tagAttr {
				continue
			}

			return ErrUnknownTag
		}

		if err := fn(v, t.Param); err != nil {
			// custom error message
			errTag, exists := tags.getByName(fmt.Sprintf("msg_%s", t.Name))
			if exists {
				errMsg := strings.Replace(errTag.Param, "{param}", t.Param, -1)
				err = errors.New(errMsg)
			}

			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errs
	}

	return nil
}

// tag represents one of the tag items
type tag struct {
	Name  string // name of the tag
	Param string // parameter to send to the validation function
}

// tagList is a list of tags
type tagList []tag

// getByName returns tag with passed name
func (tl tagList) getByName(name string) (tag, bool) {
	for _, t := range tl {
		if t.Name == name {
			return t, true
		}
	}

	return tag{}, false
}

// parseTags parses all individual tags found within a struct tag.
// TODO: caching?
func (mv *Validator) parseTags(t string) (tagList, error) {
	match := tagRegexp.FindAllStringSubmatch(t, -1)

	tags := make(tagList, 0)
	for _, group := range match {
		tg := tag{}
		tg.Name = strings.Trim(group[1], " ")

		if tg.Name == "" {
			return tagList{}, ErrUnknownTag
		}

		if len(group) > 2 {
			tg.Param = strings.Trim(group[2], " ")
		}

		tags = append(tags, tg)

	}

	return tags, nil
}
