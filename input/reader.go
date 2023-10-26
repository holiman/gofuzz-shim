package input

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"reflect"
)

// Source takes a byteslice, and arguments can be pulled from it.
type Source struct {
	data      *bytes.Reader
	exhausted bool
}

func NewSource(data []byte) *Source {
	return &Source{bytes.NewReader(data), false}
}

// IsExhausted returns true if we tried to read more data than this source
// could deliver.
func (s *Source) IsExhausted() bool {
	return s.exhausted
}

// Len returns the number of bytes of the unread portion of the data.
func (s *Source) Len() int {
	return s.data.Len()
}

// Remaining returns all remaining data in the source
func (s *Source) Remaining() []byte {
	buf := make([]byte, s.data.Len())
	s.data.Read(buf) // todo perhaps deliver via reference instead of copying via Read
	return buf
}

// Bytes returns size bytes of data.
func (s *Source) Bytes(size int) []byte {
	buf := make([]byte, size)
	_, err := s.data.Read(buf) // todo perhaps deliver via reference instead of copying via Read
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		s.exhausted = true
	}
	return buf
}

// readInt reads a signed integer from the source
func (s *Source) readInt(num reflect.Kind) int64 {
	var err error
	var ret int64
	switch num {
	case reflect.Int8:
		v := int8(0)
		err = binary.Read(s.data, binary.BigEndian, &v)
		ret = int64(v)
	case reflect.Int16:
		v := int16(0)
		err = binary.Read(s.data, binary.BigEndian, &v)
		ret = int64(v)
	case reflect.Int32:
		v := int32(0)
		err = binary.Read(s.data, binary.BigEndian, &v)
		ret = int64(v)
	case reflect.Int64, reflect.Int:
		v := int64(0)
		err = binary.Read(s.data, binary.BigEndian, &v)
		ret = int64(v)
	case reflect.Slice:
		panic(1)
	default:
		panic(fmt.Sprintf("unsupported type: %v", num))
	}
	if err == nil {
		return ret
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		s.exhausted = true
	}
	// Otherwise, this is a programming error
	panic(err)
}

// readUint reads an unsigned integer from the source
func (s *Source) readUint(num reflect.Kind) uint64 {
	var err error
	var ret uint64
	switch num {
	case reflect.Uint8:
		v := uint8(0)
		err = binary.Read(s.data, binary.BigEndian, &v)
		ret = uint64(v)
	case reflect.Uint16:
		v := uint16(0)
		err = binary.Read(s.data, binary.BigEndian, &v)
		ret = uint64(v)
	case reflect.Uint32:
		v := uint32(0)
		err = binary.Read(s.data, binary.BigEndian, &v)
		ret = uint64(v)
	case reflect.Uint, reflect.Uint64:
		v := uint64(0)
		err = binary.Read(s.data, binary.BigEndian, &v)
		ret = uint64(v)
	default:
		panic(fmt.Sprintf("unsupported type: %v", num))
	}
	if err == nil {
		return ret
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		s.exhausted = true
	}
	// Otherwise, this is a programming error
	panic(err)
}

// FillAndCall fills the argument for the given ff (which is supposed to be a function),
// and then invokes the function.
// It returns 'true' if the function was invoked. A return-value of false means
// that the method was not invoked: probably because of insufficient input.
func (s *Source) FillAndCall(ff any, arg0 reflect.Value) (ok bool) {
	fn := reflect.ValueOf(ff)
	method := fn.Type()
	if method.Kind() != reflect.Func {
		panic(fmt.Sprintf("wrong type: %T", ff))
	}
	args := make([]reflect.Value, method.NumIn())
	args[0] = arg0
	var dynamic []int
	// Fill all fixed-size arguments first, then dynamic-sized fields.
	for i := 1; i < method.NumIn(); i++ {
		v := method.In(i)
		if v.Kind() <= reflect.Float64 { // fixed-size
			args[i] = s.fillArg(v, 0)
		} else { // dynamic or panic later
			dynamic = append(dynamic, i)
		}
	}
	if s.IsExhausted() { // exit if we've exhausted the source
		return false
	}
	// Second loop to fill dynamic-sized stuff
	// For filling the dynamic fields.
	// If we have only one field, it should get all the remaining input.
	// If we have N, then,
	// 1. Read N bytes [b1, b2, b3 .. bn] .
	// 2. Let the relative weights of b determine how much of the
	//    remaining input that field n gets
	bn := s.Bytes(len(dynamic))
	sum := 0
	for _, v := range bn {
		sum += int(v)
	}
	bytesLeft := s.Len()
	for i, argNum := range dynamic {
		if i == len(dynamic)-1 { // last element, it get's all that if left
			args[argNum] = s.fillArg(method.In(argNum), s.Len())
		} else {
			var weight = (bytesLeft / len(bn))
			if sum > 0 {
				weight = (bytesLeft * int(bn[i])) / sum
			}
			args[argNum] = s.fillArg(method.In(argNum), weight)
		}
	}
	if s.IsExhausted() { // exit if we've exhausted the source
		return false
	}
	fn.Call(args)
	return true
}

func (s *Source) fillArg(v reflect.Type, max int) reflect.Value {
	newElem := reflect.New(v).Elem()
	switch k := v.Kind(); k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		newElem.SetInt(s.readInt(k))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		newElem.SetUint(s.readUint(k))
	case reflect.Float32:
		newElem.Set(reflect.ValueOf(math.Float32frombits(uint32(s.readUint(reflect.Uint32)))))
	case reflect.Float64:
		newElem.Set(reflect.ValueOf(math.Float64frombits(s.readUint(reflect.Uint64))))
	case reflect.Bool:
		newElem.Set(reflect.ValueOf(s.readUint(reflect.Uint8)&0x1 != 0))
	case reflect.String:
		newElem.SetString(string(s.Bytes(max)))
	case reflect.Slice:
		if v.Elem().Kind() == reflect.Uint8 { // []byte
			newElem.SetBytes(s.Bytes(max))
		} else {
			panic(fmt.Sprintf("unsupported type: %T", newElem.Kind))
		}
	default:
		panic(fmt.Sprintf("unsupported type: %T", newElem.Kind))
	}
	return newElem
}
