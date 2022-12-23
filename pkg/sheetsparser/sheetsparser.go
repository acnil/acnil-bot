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
	Index      int
	Field      reflect.Value
	IsReadOnly bool
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

		refs = append(refs, ref{Index: index, Field: rv.Elem().Field(i), IsReadOnly: strings.Contains(v, "ro")})
	}

	sort.Slice(refs, func(i, j int) bool { return refs[i].Index < refs[j].Index })

	out := make([]interface{}, maxRef+1)
	for _, r := range refs {
		if r.IsReadOnly {
			out[r.Index] = nil
			continue
		}
		switch r.Field.Type() {
		case reflect.TypeOf(time.Time{}):
			t := r.Field.Interface().(time.Time)
			if t.IsZero() {
				out[r.Index] = nil
				continue
			}
			out[r.Index] = t.Format(p.dateFormat())
		default:
			if f, err := strconv.ParseFloat(r.Field.String(), 64); err == nil {
				// replace . by , on float point numbers... Just calc issues
				// it is provably better to use specific types, but I'm feeling lazy
				out[r.Index] = strings.Replace(strconv.FormatFloat(f, 'f', 2, 64), ".", ",", 1)
			} else {

				out[r.Index] = r.Field.String()
			}
		}
	}

	return out, nil

}
