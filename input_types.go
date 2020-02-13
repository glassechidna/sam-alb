package main

import (
	"encoding/json"
)

type albEventProperties struct {
	Resource  string
	Alias     string
	EventName string
	Tags      map[string]json.RawMessage

	ListenerArn    json.RawMessage
	CertificateArn json.RawMessage
	Oidc           *albEventOidc      `json:",omitempty"`
	VpcConfig      *albEventVpcConfig `json:",omitempty"`
	Priority       int
	Conditions     albEventConditions
}

type albEventOidc struct {
	AuthorizationEndpoint string
	ClientId              string
	ClientSecret          string
	Issuer                string
	TokenEndpoint         string
	UserInfoEndpoint      string

	AuthenticationRequestExtraParams map[string]string `json:",omitempty"`
	OnUnauthenticatedRequest         string            `json:",omitempty"`
	Scope                            string            `json:",omitempty"`
	SessionCookieName                string            `json:",omitempty"`
	SessionTimeout                   int               `json:",omitempty"`
}

type albEventVpcConfig struct {
	SecurityGroupIds []string
	SubnetIds        []string
}

type albEventConditions struct {
	Host   json.RawMessage
	Path   json.RawMessage
	Method json.RawMessage
	Header map[string]json.RawMessage
	Ip     json.RawMessage
}
