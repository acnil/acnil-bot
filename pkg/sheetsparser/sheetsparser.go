package sheetsparser

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
)

var S = "string type"
var DefaultParser = &SheetParser{}

func Unmarshal(in []interface{}, out interface{}) error {
	return DefaultParser.Unmarshal(in, out)
}
func Marshal(in interface{}) ([]interface{}, error) {
	return DefaultParser.Marshal(in)
}

type SheetParser struct {
	DateFormat string
}

func (p *SheetParser) dateFormat() string {
	if p.DateFormat != "" {
		return p.DateFormat
	}
	return "2/1/2006"
}

func (p *SheetParser) Unmarshal(in []interface{}, out interface{}) error {
	rv := reflect.ValueOf(out)

	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return errors.New("Invalid type provided, it must be a non-nil pointer")
	}

	t := rv.Elem().Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		v, ok := field.Tag.Lookup("col")
		if !ok {
			continue
		}

		IsWriteOnly := strings.Contains(v, "wo")
		if IsWriteOnly {
			continue
		}

		fields := strings.Split(v, ",")

		index, err := strconv.Atoi(fields[0])
		if err != nil {
			return fmt.Errorf("Cannot parse column index, field %s, reason %s", field.Name, err)
		}

		elfield := rv.Elem().Field(i)
		switch elfield.Type() {
		case reflect.TypeOf(time.Time{}):
			if index >= len(in) {
				elfield.Set(reflect.ValueOf(time.Time{}))
				continue
			}
			t, err := time.Parse(p.dateFormat(), in[index].(string))
			if err != nil {
				elfield.Set(reflect.ValueOf(time.Time{}))
				continue
			}
			elfield.Set(reflect.ValueOf(t))

		case reflect.TypeOf(int(1)):
			if index >= len(in) {
				elfield.Set(reflect.ValueOf(int(0)))
				continue
			}
			v := in[index].(string)
			if len(v) == 0 {
				elfield.Set(reflect.ValueOf(int(0)))
				continue
			}
			n, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				return fmt.Errorf("couldn't parse int value, row: %#v, %s at %d, %s", in, in[index], index, err)
			}
			elfield.Set(reflect.ValueOf(int(n)))
		case reflect.TypeOf(float64(1)):
			if index >= len(in) {
				elfield.Set(reflect.ValueOf(float64(0)))
				continue
			}
			v := in[index].(string)
			if len(v) == 0 {
				elfield.Set(reflect.ValueOf(float64(0)))
				continue
			}
			n, err := strconv.ParseFloat(strings.Replace(v, ",", ".", 1), 64)
			if err != nil {
				return fmt.Errorf("couldn't parse float value, %s, %s", in[index], err)
			}
			elfield.Set(reflect.ValueOf(n))
		case reflect.TypeOf(&S):
			if index >= len(in) {
				continue
			}
			val := reflect.ValueOf(in[index])
			elfield.Set(val.Convert(elfield.Type()))

		default:
			if index >= len(in) {
				elfield.Set(reflect.ValueOf(""))
				continue
			}
			val := reflect.ValueOf(in[index])
			elfield.Set(val.Convert(elfield.Type()))
		}
	}
	return nil
}

type ref struct {
	Index       int
	Field       reflect.Value
	IsReadOnly  bool
	IsWriteOnly bool
	OmitEmpty   bool
}

func (p *SheetParser) Marshal(in interface{}) ([]interface{}, error) {
	rv := reflect.ValueOf(in)

	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return nil, errors.New("Invalid type provided, it must be a non-nil pointer")
	}

	refs := []ref{}

	maxRef := 0

	t := rv.Elem().Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		v, ok := field.Tag.Lookup("col")
		if !ok {
			continue
		}
		fields := strings.Split(v, ",")
		index, err := strconv.Atoi(fields[0])
		if err != nil {
			return nil, fmt.Errorf("Cannot parse column index, field %s, reason %s", field.Name, err)
		}
		if index > maxRef {
			maxRef = index
		}

		refs = append(refs, ref{
			Index:       index,
			Field:       rv.Elem().Field(i),
			IsReadOnly:  strings.Contains(v, "ro"),
			IsWriteOnly: strings.Contains(v, "wo"),
			OmitEmpty:   strings.Contains(v, "omitempty"),
		})
	}

	sort.Slice(refs, func(i, j int) bool {
		if refs[i].Index == refs[j].Index {
			if refs[i].IsReadOnly {
				return true
			}
			if refs[j].IsReadOnly {
				return false
			}
		}
		return refs[i].Index < refs[j].Index
	})

	out := make([]interface{}, maxRef+1)
	for _, r := range refs {
		if r.IsReadOnly {
			out[r.Index] = nil
			continue
		}
		fieldType := r.Field.Type()
		switch {
		case fieldType == reflect.TypeOf(time.Time{}):
			t := r.Field.Interface().(time.Time)
			if t.IsZero() {
				out[r.Index] = nil
				continue
			}
			out[r.Index] = t.Format(p.dateFormat())
		case fieldType == reflect.TypeOf(int(1)):
			n := r.Field.Int()
			out[r.Index] = strconv.FormatInt(n, 10)
		case fieldType == reflect.TypeOf(float64(1)):
			n := r.Field.Float()
			out[r.Index] = strings.Replace(strconv.FormatFloat(n, 'f', 2, 64), ".", ",", 1)
		case fieldType.Kind() == reflect.Pointer:
			v := r.Field.Elem()
			if v.Kind() == reflect.Invalid {
				out[r.Index] = nil
				continue
			}
			out[r.Index] = v.String()
		default:
			if r.OmitEmpty && r.Field.String() == "" {
				out[r.Index] = nil
				continue
			}
			out[r.Index] = r.Field.String()
		}
	}

	return out, nil

}
