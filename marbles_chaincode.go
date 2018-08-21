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
	} else if function == "queryByDate" {
		return t.queryByDate(stub, args)
	} else if function == "getHistoryByDate" {
		return t.getHistoryByDate(stub, args)
	} else if function == "getDeviceList" {
		return t.getDeviceList(stub, args)
	} else if function == "queryLocation" {
		return t.queryLocation(stub, args)
	} else if function == "saveDevice" {
		return t.saveDevice(stub, args)
	}

	return shim.Error("Invalid function name for 'invoke'")
}

// saveNewEvent stores the event on the ledger. For each device
// it will override the current state with the new one
func (t *SimpleAsset) saveNewEvent(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	if len(args) != 5 {
		return shim.Error("incorrect arguments. Expecting full event details. Expecting 5 args")
	}

	displayName := strings.ToLower(args[0])
	deviceID := strings.ToLower(args[1])
	locationID := strings.ToLower(args[2])
	value := strings.ToLower(args[3])
	time := strings.ToLower(args[4])
	date := strings.Replace(time, "-", "", -1)
	date = strings.Split(date, "t")[0]
	//Building the event json string manually without struct marshalling
	eventJSONasString := `{"docType":"Event",  "displayName": "` + displayName + `",
	 "deviceId": "` + deviceID + `", "locationId": "` + locationID + `",
	 "value": "` + value + `", "time": "` + time + `", "date": "` + date + `"}`
	eventJSONasBytes := []byte(eventJSONasString)

	arr := []string{deviceID, time}
	myCompositeKey, err := stub.CreateCompositeKey("combined", arr)
	if err != nil {
		return shim.Error("Failed to set composite key")
	}
	err = stub.PutState(myCompositeKey, eventJSONasBytes)
	if err != nil {
		return shim.Error("Failed to set asset")
	}
	return shim.Success([]byte(displayName))
}

func (t *SimpleAsset) saveDevice(stub shim.ChaincodeStubInterface, args []string) peer.Response {

	locationID := strings.ToLower(args[0])
	deviceIdList := args[1]
	deviceNameList := args[2]

	//Building the event json string manually without struct marshalling
	deviceJSONasString := `{"docType":"deviceId",  "ids": "` + deviceIdList + `", "names": "` + deviceNameList + `"}`
	deviceJSONasBytes := []byte(deviceJSONasString)
	err := stub.PutState(locationID, deviceJSONasBytes)
	if err != nil {
		return shim.Error("Failed to set asset")
	}

	return shim.Success([]byte(locationID))
}

// main function starts up the chaincode in the container during instantiate
func main() {
	if err := shim.Start(new(SimpleAsset)); err != nil {
		fmt.Printf("Error starting SimpleAsset chaincode: %s", err)
	}
}

// getQueryResultForQueryString retrieves the data from couchdb
// for rich queries passed as a string
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

// queryLocation creates a rich query to query the location using locationId.
// It retrieve all the devices and their last states for that location.
func (t *SimpleAsset) queryLocation(stub shim.ChaincodeStubInterface, args []string) peer.Response {

	if len(args) < 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	locationId := args[0]

	// queryString := fmt.Sprintf("{\r\n    \"selector\": {\r\n        \"docType\": \"Event\",\r\n        \"locationId\": \"%s\"\r\n    },\r\n    \"fields\": [\"displayName\", \"value\",\"time\",\"date\"]\r\n}", locationId)
	queryString := fmt.Sprintf("{\r\n    \"selector\": {\r\n        \"docType\": \"EventLess\",\r\n        \"locationId\": \"%s\"\r\n    },\r\n    \"fields\": [\"displayName\", \"value\",\"time\"]\r\n}", locationId)

	queryResults, err := getQueryResultForQueryString(stub, queryString)
	if err != nil {
		return shim.Error(err.Error())
	}
	queryResultsString := strings.Replace(string(queryResults), "\u0000", "||", -1)
	// queryResultsString, err = url.QueryUnescape(string(queryResultsString))

	if err != nil {
		return shim.Error(err.Error())
	}
	queryResults = []byte(queryResultsString)
	return shim.Success(queryResults)
}

// queryByTime creates a rich query to query using locationId, deviceId and date.
// It retrieves all the history of the device for particular date.
func (t *SimpleAsset) queryByDate(stub shim.ChaincodeStubInterface, args []string) peer.Response {

	if len(args) < 3 {
		return shim.Error("Incorrect number of arguments. Expecting 3")
	}

	locationId := args[0]
	deviceId := args[1]
	date := args[2]
	queryString := fmt.Sprintf("{\r\n    \"selector\": {\r\n        \"docType\": \"Event\",\r\n        \"locationId\": \"%s\",\r\n        \"deviceId\": \"%s\",\r\n        \"date\": \"%s\"\r\n    },\r\n    \"fields\": [\"value\",\"time\"]\r\n}", locationId, deviceId, date)

	queryResults, err := getQueryResultForQueryString(stub, queryString)
	queryResultsString := strings.Replace(string(queryResults), "\u0000", "||", -1)
	if err != nil {
		return shim.Error(err.Error())
	}
	queryResults = []byte(queryResultsString)
	return shim.Success(queryResults)
}

func (t *SimpleAsset) getHistoryByDate(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	if len(args) != 2 {
		return shim.Error("incorrect arguments. Expecting 2")
	}
	deviceId := args[0]
	date := args[1]
	arr := []string{deviceId, date}
	myCompositeKey, err := stub.CreateCompositeKey("combined", arr)
	if err != nil {
		return shim.Error("Failed to set composite key")
	}
	resultsIterator, err := stub.GetHistoryForKey(myCompositeKey)
	if err != nil {
		return shim.Error(err.Error())
	}
	defer resultsIterator.Close()
	// buffer is a JSON array containing historic values for the marble
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

func (t *SimpleAsset) getDeviceList(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	var jsonResp string
	if len(args) != 1 {
		return shim.Error("incorrect arguments. Expecting 1")
	}
	locationId := args[0]
	valueAsBytes, err := stub.GetState(locationId)
	if err != nil {
		jsonResp = "{\"Error\":\"Failed to get state for " + locationId + "\"}"
		return shim.Error(jsonResp)
	}
	if valueAsBytes == nil {
		jsonResp = "{\"Error\":\"Transaction does not exist: " + locationId + "\"}"
		return shim.Error(jsonResp)
	}
	return shim.Success(valueAsBytes)
}
