package main

import "encoding/json"

type MacroInput struct {
	Region      string
	AccountId   string
	TransformId string
	RequestId   string
	Fragment    json.RawMessage
	Params      json.RawMessage
	Values      map[string]json.RawMessage `json:"templateParameterValues"`
}

type MacroOutput struct {
	Status    string          `json:"status"`
	RequestId string          `json:"requestId"`
	Fragment  json.RawMessage `json:"fragment"`
}

const MacroOutputStatusSuccess = "success"
const MacroOutputStatusFailure = "failure"

