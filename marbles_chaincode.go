/**
 *  Blockchain Event Logger
 *
 *  Copyright 2018 Xooa
 *
 *  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except
 *  in compliance with the License. You may obtain a copy of the License at:
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software distributed under the License is distributed
 *  on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License
 *  for the specific language governing permissions and limitations under the License.
 */
/*
 * Original source via IBM Corp:
 *  https://hyperledger-fabric.readthedocs.io/en/release-1.2/chaincode4ade.html#pulling-it-all-together
 *
 * Modifications from: Arisht Jain:
 *  https://github.com/xooa/smartThings-xooa
 *
 * Changes:
 *  Logs to Xooa blockchain platform from SmartThings instead from user
 */

package main

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/protos/peer"
)

// SimpleAsset implements a simple chaincode to manage an asset
type SimpleAsset struct {
}

// Init is called during chaincode instantiation to initialize any
// data. Note that chaincode upgrade also calls this function to reset
// or to migrate data.
func (t *SimpleAsset) Init(stub shim.ChaincodeStubInterface) peer.Response {
	return shim.Success(nil)
}

// Invoke is called per transaction on the chaincode. Each transaction is
// either updating the state or retreiving the state created by Init function.
func (t *SimpleAsset) Invoke(stub shim.ChaincodeStubInterface) peer.Response {
	// Extract the function and args from the transaction proposal
	function, args := stub.GetFunctionAndParameters()

	if function == "saveNewEvent" {
		return t.saveNewEvent(stub, args)
	} else if function == "getDeviceLastEvent" {
		return t.getDeviceLastEvent(stub)
	} else if function == "getHistoryForDevice" {
		return t.getHistoryForDevice(stub, args)
	} else if function == "getEvent" {
		return t.getEvent(stub, args)
	} else if function == "queryLocation" {
		return t.queryLocation(stub, args)
	} else if function == "queryDevice" {
		return t.queryDevice(stub, args)
	}

	return shim.Error("Invalid function name for 'invoke'")
}

// saveNewEvent stores the event on the ledger. For each device
// it will override the current state with the new one
func (t *SimpleAsset) saveNewEvent(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	if len(args) != 17 {
		return shim.Error("incorrect arguments. Expecting full event details")
	}

	displayName := strings.ToLower(args[0])
	device := strings.ToLower(args[1])
	isStateChange := strings.ToLower(args[2])
	id := strings.ToLower(args[3])
	description := strings.ToLower(args[4])
	descriptionText := strings.ToLower(args[5])
	installedSmartAppID := strings.ToLower(args[6])
	isDigital := strings.ToLower(args[7])
	isPhysical := strings.ToLower(args[8])
	deviceID := strings.ToLower(args[9])
	location := strings.ToLower(args[10])
	locationID := strings.ToLower(args[11])
	source := strings.ToLower(args[12])
	unit := strings.ToLower(args[13])
	value := strings.ToLower(args[14])
	name := strings.ToLower(args[15])
	time := strings.ToLower(args[16])

	//Building the event json string manually without struct marshalling
	eventJSONasString := `{"docType":"Event",  "displayName": "` + displayName + `",
	 "device": "` + device + `", "isStateChange": "` + isStateChange + `",
	 "id": "` + id + `", "description": "` + description + `",
	 "descriptionText": "` + descriptionText + `", "installedSmartAppId": "` + installedSmartAppID + `",
	 "isDigital": "` + isDigital + `", "isPhysical": "` + isPhysical + `", "deviceId": "` + deviceID + `",
	 "location": "` + location + `", "locationId": "` + locationID + `", "source": "` + source + `",
	 "unit": "` + unit + `", "value": "` + value + `", "name": "` + name + `", "time": "` + time + `"}`
	eventJSONasBytes := []byte(eventJSONasString)

	err1 := stub.PutState(deviceID, eventJSONasBytes)
	if err1 != nil {
		return shim.Error("Failed to set asset")
	}
	return shim.Success([]byte(device))
}

// getEvent returns the value of the specified deviceId
func (t *SimpleAsset) getEvent(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	var userID, jsonResp string
	var err error

	if len(args) != 1 {
		return shim.Error("Incorrect arguments. Expecting a key")
	}

	valueAsBytes, err := stub.GetState(args[0])
	if err != nil {
		jsonResp = "{\"Error\":\"Failed to get state for " + userID + "\"}"
		return shim.Error(jsonResp)
	}
	if valueAsBytes == nil {
		jsonResp = "{\"Error\":\"Transaction does not exist: " + userID + "\"}"
		return shim.Error(jsonResp)
	}
	return shim.Success(valueAsBytes)
}

// Gets the last transactions for all unique keys i.e., all devices
func (t *SimpleAsset) getDeviceLastEvent(stub shim.ChaincodeStubInterface) peer.Response {
	resultsIterator, err := stub.GetStateByRange("", "")
	if err != nil {
		return shim.Error("Failed to get data " + err.Error())
	}
	defer resultsIterator.Close()

	var buffer bytes.Buffer
	buffer.WriteString("[")

	bArrayMemberAlreadyWritten := false

	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return shim.Error(err.Error())
		}

		if bArrayMemberAlreadyWritten == true {
			buffer.WriteString(",")
		}
		buffer.WriteString("{\"DeviceId\":")
		buffer.WriteString("\"")
		buffer.WriteString(queryResponse.Key)
		buffer.WriteString("\"")

		buffer.WriteString(", \"Record\":")
		buffer.WriteString(string(queryResponse.Value))
		buffer.WriteString("}")
		bArrayMemberAlreadyWritten = true
	}
	buffer.WriteString("]")

	return shim.Success(buffer.Bytes())
}

// For a given device, return its historical data
func (t *SimpleAsset) getHistoryForDevice(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	if len(args) != 1 {
		return shim.Error("Expected argument lenght doesnt match")
	}

	resultsIterator, err := stub.GetHistoryForKey(args[0])
	if err != nil {
		return shim.Error("Failed to get data " + err.Error())
	}
	defer resultsIterator.Close()

	var buffer bytes.Buffer
	buffer.WriteString("[")

	bArrayMemberAlreadyWritten := false

	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return shim.Error(err.Error())
		}

		if bArrayMemberAlreadyWritten == true {
			buffer.WriteString(",")
		}
		buffer.WriteString("{\"TxId\":")
		buffer.WriteString("\"")
		buffer.WriteString(queryResponse.TxId)
		buffer.WriteString("\"")

		buffer.WriteString(", \"Timestamp\":")
		buffer.WriteString("\"")
		buffer.WriteString(time.Unix(queryResponse.Timestamp.Seconds, int64(queryResponse.Timestamp.Nanos)).String())
		buffer.WriteString("\"")

		buffer.WriteString(", \"Record\":")
		buffer.WriteString(string(queryResponse.Value))
		buffer.WriteString("}")
		bArrayMemberAlreadyWritten = true
	}
	buffer.WriteString("]")

	return shim.Success(buffer.Bytes())
}

// main function starts up the chaincode in the container during instantiate
func main() {
	if err := shim.Start(new(SimpleAsset)); err != nil {
		fmt.Printf("Error starting SimpleAsset chaincode: %s", err)
	}
}

func getQueryResultForQueryString(stub shim.ChaincodeStubInterface, queryString string) ([]byte, error) {

	fmt.Printf("- getQueryResultForQueryString queryString:\n%s\n", queryString)

	resultsIterator, err := stub.GetQueryResult(queryString)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	// buffer is a JSON array containing QueryRecords
	var buffer bytes.Buffer
	buffer.WriteString("[")

	bArrayMemberAlreadyWritten := false
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}
		// Add a comma before array members, suppress it for the first array member
		if bArrayMemberAlreadyWritten == true {
			buffer.WriteString(",")
		}
		buffer.WriteString("{\"Key\":")
		buffer.WriteString("\"")
		buffer.WriteString(queryResponse.Key)
		buffer.WriteString("\"")

		buffer.WriteString(", \"Record\":")
		// Record is a JSON object, so we write as-is
		buffer.WriteString(string(queryResponse.Value))
		buffer.WriteString("}")
		bArrayMemberAlreadyWritten = true
	}
	buffer.WriteString("]")

	fmt.Printf("- getQueryResultForQueryString queryResult:\n%s\n", buffer.String())

	return buffer.Bytes(), nil
}

func (t *SimpleAsset) queryLocation(stub shim.ChaincodeStubInterface, args []string) peer.Response {

	//   0
	// "bob"
	if len(args) < 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	locationId := args[0]

	queryString := fmt.Sprintf("{\r\n    \"selector\": {\r\n        \"docType\": \"Event\",\r\n        \"locationId\": \"%s\"\r\n    },\r\n    \"fields\": [\"displayName\", \"deviceId\", \"value\",\"time\"]\r\n}", locationId)

	queryResults, err := getQueryResultForQueryString(stub, queryString)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(queryResults)
}

func (t *SimpleAsset) queryDevice(stub shim.ChaincodeStubInterface, args []string) peer.Response {

	//   0
	// "bob"
	if len(args) < 2 {
		return shim.Error("Incorrect number of arguments. Expecting 2")
	}

	locationId := args[0]
	deviceId := args[1]
	queryString := fmt.Sprintf("{\"selector\":{\"docType\":\"Event\",\"locationId\":\"%s\",\"deviceId\":\"%s\"}}", locationId, deviceId)

	queryResults, err := getQueryResultForQueryString(stub, queryString)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(queryResults)
}
