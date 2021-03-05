//
// Copyright (c) 2020 Intel Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package transforms

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/edgexfoundry/go-mod-core-contracts/clients"
	"github.com/edgexfoundry/go-mod-core-contracts/models"
	"github.com/epcom-hdxt/app-functions-sdk-go/appcontext"
)

// Conversion houses various built in conversion transforms (XML, JSON, CSV)
type Conversion struct {
}

// NewConversion creates, initializes and returns a new instance of Conversion
func NewConversion() Conversion {
	return Conversion{}
}

// TransformToXML transforms an EdgeX event to XML.
// It will return an error and stop the pipeline if a non-edgex event is received or if no data is received.
func (f Conversion) TransformToXML(edgexcontext *appcontext.Context, params ...interface{}) (continuePipeline bool, stringType interface{}) {
	if len(params) < 1 {
		return false, errors.New("No Event Received")
	}
	edgexcontext.LoggingClient.Debug("Transforming to XML")
	if event, ok := params[0].(models.Event); ok {
		xml, err := event.ToXML()
		if err != nil {
			return false, fmt.Errorf("unable to marshal Event to XML: %s", err.Error())
		}
		edgexcontext.ResponseContentType = clients.ContentTypeXML
		return true, xml
	}
	return false, errors.New("Unexpected type received")
}

// TransformToJSON transforms an EdgeX event to JSON.
// It will return an error and stop the pipeline if a non-edgex event is received or if no data is received.
func (f Conversion) TransformToJSON(edgexcontext *appcontext.Context, params ...interface{}) (continuePipeline bool, stringType interface{}) {
	if len(params) < 1 {
		return false, errors.New("No Event Received")
	}
	edgexcontext.LoggingClient.Debug("Transforming to JSON")
	if result, ok := params[0].(models.Event); ok {
		b, err := json.Marshal(result)
		if err != nil {
			// LoggingClient.Error(fmt.Sprintf("Error parsing JSON. Error: %s", err.Error()))
			return false, errors.New("Error marshalling JSON")
		}
		edgexcontext.ResponseContentType = clients.ContentTypeJSON
		// should we return a byte[] or string?
		// return b
		return true, string(b)
	}
	return false, errors.New("Unexpected type received")
}

// TransformToJSON transforms an EdgeX event to JSON.
// It will return an error and stop the pipeline if a non-edgex event is received or if no data is received.
func (f Conversion) CustomTransformToJson(edgexcontext *appcontext.Context, params ...interface{}) (continuePipeline bool, stringType interface{}) {
	if len(params) < 1 {
		return false, errors.New("No Event Received")
	}
	edgexcontext.LoggingClient.Debug("Transforming to JSON")
	if result, ok := params[0].(models.Event); ok {
		readings := result.Readings

		var build strings.Builder
		build.WriteString("[{\"data\":[{")
		var dtype = "value"
		// var shiftFlagStr = ""
		//Todo from db
		shiftflag := []string{"w1", "w2", "w3", "f1", "f2", "p1", "p2", "p3", "rs"}
		shiftFlagStr := strings.Join(shiftflag, ",")
		flagmap := CreateShiftFlagMap(shiftflag)
		for i, item := range readings {

			// if item.Name == "w1"  ... item.Name == "p2"
			if _, ok := flagmap[item.Name]; ok {
				build.WriteString("\"" + item.Name + "\":[{")
				dtype = "bit"
				shiftmap := CreateShiftMap(item.Value)

				for k := 0; k < 16; k++ {
					value, ok := shiftmap["b"+strconv.Itoa(k)]
					if ok {
						build.WriteString("\"b" + strconv.Itoa(k) + "\":" + strconv.Itoa(value) + "")
					} else {
						return false, errors.New("Unexpected type received")
					}

					build.WriteString(",")

				}
				build.WriteString("\"value\":\"" + item.Value + "\"}]")
			} else {
				if item.ValueType == "Float32" {
					decodeBytes, _ := base64.StdEncoding.DecodeString(item.Value)
					build.WriteString("\"" + item.Name + "\":\"" + FloatToString(ByteToFloat32(decodeBytes)) + "\"")
				} else {
					build.WriteString("\"" + item.Name + "\":\"" + item.Value + "\"")
				}
			}
			if i < len(readings)-1 {
				build.WriteString(",")
			}

		}
		if dtype == "bit" {
			build.WriteString(",\"shiftFlagStr\":\"" + shiftFlagStr + "\"}]")

		} else {
			build.WriteString("}]")

		}

		build.WriteString(",\"device\":\"" + result.Device + "\",\"created\":\"" + strconv.FormatInt(result.Created, 10) + "\",\"origin\":\"" + strconv.FormatInt(result.Origin, 10) + "\",\"dtype\":\"" + dtype + "\"}]")

		edgexcontext.ResponseContentType = clients.ContentTypeJSON
		fmt.Println(build.String())
		return true, build.String()
	}
	return false, errors.New("Unexpected type received")
}

func CreateShiftFlagMap(flag []string) map[string]string {
	var shiftflagmap map[string]string
	shiftflagmap = make(map[string]string)
	for _, s := range flag {
		shiftflagmap[s] = s
	}
	return shiftflagmap

}

func CreateShiftMap(value string) map[string]int {
	var shiftmap map[string]int
	shiftmap = make(map[string]int)
	itemValue, _ := strconv.Atoi(value)
	shiftmap["b0"] = itemValue & 0x1
	shiftmap["b1"] = (itemValue & 0x2) / 0x2
	shiftmap["b2"] = (itemValue & 0x4) / 0x4
	shiftmap["b3"] = (itemValue & 0x8) / 0x8
	shiftmap["b4"] = (itemValue & 0x10) / 0x10
	shiftmap["b5"] = (itemValue & 0x20) / 0x20
	shiftmap["b6"] = (itemValue & 0x40) / 0x40
	shiftmap["b7"] = (itemValue & 0x80) / 0x80
	shiftmap["b8"] = (itemValue & 0x100) / 0x100
	shiftmap["b9"] = (itemValue & 0x200) / 0x200
	shiftmap["b10"] = (itemValue & 0x400) / 0x400
	shiftmap["b11"] = (itemValue & 0x800) / 0x800
	shiftmap["b12"] = (itemValue & 0x1000) / 0x1000
	shiftmap["b13"] = (itemValue & 0x2000) / 0x2000
	shiftmap["b14"] = (itemValue & 0x4000) / 0x4000
	shiftmap["b15"] = (itemValue & 0x8000) / 0x8000

	return shiftmap
}

func Float32ToByte(float float32) []byte {
	bits := math.Float32bits(float)
	bytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(bytes, bits)

	return bytes
}
func FloatToString(input_num float32) string {
	// to convert a float number to a string
	return strconv.FormatFloat(float64(input_num), 'f', 10, 64)
}

func ByteToFloat32(bytes []byte) float32 {
	bits := binary.BigEndian.Uint32(bytes)

	return math.Float32frombits(bits)
}
func Float64ToByte(float float64) []byte {
	bits := math.Float64bits(float)
	bytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(bytes, bits)

	return bytes
}

func ByteToFloat64(bytes []byte) float64 {
	bits := binary.LittleEndian.Uint64(bytes)

	return math.Float64frombits(bits)
}
func BytesToInt(b []byte) int {
	bytesBuffer := bytes.NewBuffer(b)

	var x int32
	binary.Read(bytesBuffer, binary.BigEndian, &x)

	return int(x)
}
