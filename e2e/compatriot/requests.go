package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

type RPCRequest struct {
	Jsonrpc string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  []any  `json:"params"`
	Id      int64  `json:"id"`
}

type ResponseErr struct {
	Code    int64  `json:"code"`
	Message string `json:"message"`
}

type RPCResponse struct {
	Jsonrpc string      `json:"jsonrpc"`
	Id      int64       `json:"id"`
	Result  any         `json:"result"`
	Err     ResponseErr `json:"error"`
}

type RPCOutput struct {
	Method   string      `json:"method"`
	Response RPCResponse `json:"response"`
}

// Query loads prexisting JSON-RPC calls from a file and queries the chain
func Query(outputFile string) error {
	calls := generateQueries() // generates the queries according to the chain setup

	var output []RPCOutput
	for i := 0; i < len(calls); i++ {
		result, err := call(calls[i])
		if err != nil {
			return fmt.Errorf("Query: An error occurred %v when calling", err)
		}
		output = append(output, result)
	}

	// add the results to a file and format
	content, err := Marshal(output)
	if err != nil {
		return fmt.Errorf("Query: An error occurred %v when marshalling output", err)
	}

	if err = os.WriteFile("./"+outputFile, content, 0644); err != nil {
		return fmt.Errorf("call: An error occurred %v when writing output", err)
	}

	fmt.Println("finished querying")
	return nil
}

// call makes a JSON-RPC call to the chain and saves the results
func call(postRequest RPCRequest) (RPCOutput, error) {
	postBody, _ := json.Marshal(postRequest)
	buffer := bytes.NewBuffer(postBody)

	body, err := makeRequest(POLARIS_RPC, buffer)
	if err != nil {
		return RPCOutput{}, fmt.Errorf("call: An error occurred %v when making the request", err)
	}
	var response RPCResponse
	json.Unmarshal([]byte(body), &response)

	return RPCOutput{Method: postRequest.Method, Response: response}, nil
}

// makeRequest makes the actual HTTP request to the chain
func makeRequest(rpc string, postBuffer *bytes.Buffer) (string, error) {
	response, err := http.Post(rpc, "application/json", postBuffer)
	if err != nil {
		return "", fmt.Errorf("makeRequest: An Error Occured %v when posting", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("makeRequest: An Error Occured %v when reading response", err)
	}
	return string(body), nil
}

// Marshal marshals the output slice to JSON
func Marshal(output []RPCOutput) ([]byte, error) {
	jsonOutput, err := json.MarshalIndent(output, "", "    ")
	if err != nil {
		return nil, fmt.Errorf("Marshal: An error occurred %v trying to marshal data", err)
	}
	return jsonOutput, nil
}
