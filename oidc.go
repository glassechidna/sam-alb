package main

import (
	"encoding/json"
	"github.com/pkg/errors"
	"io/ioutil"
	"net/http"
	"strings"
)

var oidcClient = http.DefaultClient

func oidcAction(input *albEventOidc) (*ActionProperty, error) {
	ae := input.AuthorizationEndpoint
	te := input.TokenEndpoint
	uie := input.UserInfoEndpoint
	if len(ae) == 0 || len(te) == 0 || len(uie) == 0 {
		doc, err := discover(input.Issuer)
		if err != nil {
			return nil, err
		}

		if len(ae) == 0 {
			ae = doc.AuthorizationEndpoint
		}

		if len(te) == 0 {
			te = doc.TokenEndpoint
		}

		if len(uie) == 0 {
			uie = doc.UserInfoEndpoint
		}
	}

	return &ActionProperty{
		Order: 1,
		Type:  "authenticate-oidc",
		Oidc: &albEventOidc{
			Issuer:                input.Issuer,
			ClientId:              input.ClientId,
			ClientSecret:          input.ClientSecret,
			AuthorizationEndpoint: ae,
			TokenEndpoint:         te,
			UserInfoEndpoint:      uie,

			// optional
			AuthenticationRequestExtraParams: nil,
			OnUnauthenticatedRequest:         "",
			Scope:                            "",
			SessionCookieName:                "",
			SessionTimeout:                   0,
		},
	}, nil
}

type discoveryDocument struct {
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	UserInfoEndpoint      string `json:"userinfo_endpoint"`
}

func discover(issuer string) (*discoveryDocument, error) {
	discoveryUrl := strings.TrimSuffix(issuer, "/") + "/.well-known/openid-configuration"
	resp, err := oidcClient.Get(discoveryUrl)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	doc := discoveryDocument{}
	err = json.Unmarshal(body, &doc)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &doc, nil
}

