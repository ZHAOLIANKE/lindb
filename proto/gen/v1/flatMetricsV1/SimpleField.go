// Licensed to LinDB under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. LinDB licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

// Code generated by the FlatBuffers compiler. DO NOT EDIT.

package flatMetricsV1

import (
	flatbuffers "github.com/google/flatbuffers/go"
)

type SimpleField struct {
	_tab flatbuffers.Table
}

func GetRootAsSimpleField(buf []byte, offset flatbuffers.UOffsetT) *SimpleField {
	n := flatbuffers.GetUOffsetT(buf[offset:])
	x := &SimpleField{}
	x.Init(buf, n+offset)
	return x
}

func (rcv *SimpleField) Init(buf []byte, i flatbuffers.UOffsetT) {
	rcv._tab.Bytes = buf
	rcv._tab.Pos = i
}

func (rcv *SimpleField) Table() flatbuffers.Table {
	return rcv._tab
}

func (rcv *SimpleField) Name() []byte {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(4))
	if o != 0 {
		return rcv._tab.ByteVector(o + rcv._tab.Pos)
	}
	return nil
}

func (rcv *SimpleField) Type() SimpleFieldType {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(6))
	if o != 0 {
		return SimpleFieldType(rcv._tab.GetInt8(o + rcv._tab.Pos))
	}
	return 0
}

func (rcv *SimpleField) MutateType(n SimpleFieldType) bool {
	return rcv._tab.MutateInt8Slot(6, int8(n))
}

func (rcv *SimpleField) Value() float64 {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(8))
	if o != 0 {
		return rcv._tab.GetFloat64(o + rcv._tab.Pos)
	}
	return 0.0
}

func (rcv *SimpleField) MutateValue(n float64) bool {
	return rcv._tab.MutateFloat64Slot(8, n)
}

func (rcv *SimpleField) Exemplars(obj *Exemplar, j int) bool {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(10))
	if o != 0 {
		x := rcv._tab.Vector(o)
		x += flatbuffers.UOffsetT(j) * 4
		x = rcv._tab.Indirect(x)
		obj.Init(rcv._tab.Bytes, x)
		return true
	}
	return false
}

func (rcv *SimpleField) ExemplarsLength() int {
	o := flatbuffers.UOffsetT(rcv._tab.Offset(10))
	if o != 0 {
		return rcv._tab.VectorLen(o)
	}
	return 0
}

func SimpleFieldStart(builder *flatbuffers.Builder) {
	builder.StartObject(4)
}
func SimpleFieldAddName(builder *flatbuffers.Builder, name flatbuffers.UOffsetT) {
	builder.PrependUOffsetTSlot(0, flatbuffers.UOffsetT(name), 0)
}
func SimpleFieldAddType(builder *flatbuffers.Builder, type_ SimpleFieldType) {
	builder.PrependInt8Slot(1, int8(type_), 0)
}
func SimpleFieldAddValue(builder *flatbuffers.Builder, value float64) {
	builder.PrependFloat64Slot(2, value, 0.0)
}
func SimpleFieldAddExemplars(builder *flatbuffers.Builder, exemplars flatbuffers.UOffsetT) {
	builder.PrependUOffsetTSlot(3, flatbuffers.UOffsetT(exemplars), 0)
}
func SimpleFieldStartExemplarsVector(builder *flatbuffers.Builder, numElems int) flatbuffers.UOffsetT {
	return builder.StartVector(4, numElems, 4)
}
func SimpleFieldEnd(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	return builder.EndObject()
}
