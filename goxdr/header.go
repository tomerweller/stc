
package main

var header string = `
// begin boilerplate

import "fmt"
import "math"
import "io"

var _ = fmt.Sprintf
type _ = io.Writer

type XdrError string
func (v XdrError) Error() string { return string(v) }
func xdrPanic(s string, args ...interface{}) {
	panic(XdrError(fmt.Sprintf(s, args...)))
}

const (
	TRUE = true
	FALSE = false
)

type XdrVoid = struct{}

type XdrNum32 interface {
	GetU32() uint32
	SetU32(uint32)
	XdrPointer() interface{}
	XdrValue() interface{}
}

type XdrBool bool
func (v *XdrBool) GetU32() uint32 {
	if *v {
		return 1
	}
	return 0
}
func (v *XdrBool) SetU32(nv uint32) {
	switch nv {
	case 0:
		*v = false
	case 1:
		*v = true
	}
	xdrPanic("Bool must be 0 or 1")
}
func (v *XdrBool) XdrPointer() interface{} { return (*bool)(v) }
func (v *XdrBool) XdrValue() interface{} { return bool(*v) }

type XdrInt32 int32
func (v *XdrInt32) GetU32() uint32 { return uint32(*v) }
func (v *XdrInt32) SetU32(nv uint32) { *v = XdrInt32(nv) }
func (v *XdrInt32) XdrPointer() interface{} { return (*int32)(v) }
func (v *XdrInt32) XdrValue() interface{} { return int32(*v) }

type XdrUint32 uint32
func (v *XdrUint32) GetU32() uint32 { return uint32(*v) }
func (v *XdrUint32) SetU32(nv uint32) { *v = XdrUint32(nv) }
func (v *XdrUint32) XdrPointer() interface{} { return (*uint32)(v) }
func (v *XdrUint32) XdrValue() interface{} { return uint32(*v) }

type XdrFloat32 float32
func (v *XdrFloat32) GetU32() uint32 { return math.Float32bits(float32(*v)) }
func (v *XdrFloat32) SetU32(nv uint32) {
	*v = XdrFloat32(math.Float32frombits(nv))
}
func (v *XdrFloat32) XdrPointer() interface{} { return (*float32)(v) }
func (v *XdrFloat32) XdrValue() interface{} { return float32(*v) }

type XdrNum64 interface {
	GetU64() uint64
	SetU64(uint64)
	XdrPointer() interface{}
	XdrValue() interface{}
}

type XdrInt64 int64
func (v *XdrInt64) GetU64() uint64 { return uint64(*v) }
func (v *XdrInt64) SetU64(nv uint64) { *v = XdrInt64(nv) }
func (v *XdrInt64) XdrPointer() interface{} { return (*int64)(v) }
func (v *XdrInt64) XdrValue() interface{} { return int64(*v) }

type XdrUint64 uint64
func (v *XdrUint64) GetU64() uint64 { return uint64(*v) }
func (v *XdrUint64) SetU64(nv uint64) { *v = XdrUint64(nv) }
func (v *XdrUint64) XdrPointer() interface{} { return (*uint64)(v) }
func (v *XdrUint64) XdrValue() interface{} { return uint64(*v) }

type XdrFloat64 float64
func (v *XdrFloat64) GetU64() uint64 { return math.Float64bits(float64(*v)) }
func (v *XdrFloat64) SetU64(nv uint64) {
	*v = XdrFloat64(math.Float64frombits(nv))
}
func (v *XdrFloat64) XdrPointer() interface{} { return (*float64)(v) }
func (v *XdrFloat64) XdrValue() interface{} { return float64(*v) }

type XdrBytes interface {
	GetByteSlice() []byte
}
type XdrVariableBytes interface {
	XdrBound() uint32
	SetByteSlice([]byte)
}

type XdrOpaqueArray []byte
func (v *XdrOpaqueArray) GetByteSlice() []byte { return ([]byte)(*v) }

// end boilerplate` + "\n"
