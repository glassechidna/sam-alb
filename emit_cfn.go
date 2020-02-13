package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"sort"
)

type stringOrRef struct {
	Value string
	Ref   string
}

func (s stringOrRef) MarshalJSON() ([]byte, error) {
	if len(s.Value) > 0 {
		return json.Marshal(s.Value)
	} else {
		return json.Marshal(map[string]string{"Ref": s.Ref})
	}
}

type rule struct {
	Priority    int
	ListenerArn json.RawMessage
	Actions     []ActionProperty
	Conditions  []cfnCondition
}

type keyvaluePair struct {
	Key   string
	Value string
}

type cfnCondition struct {
	Field                   string
	HostHeaderConfig        *jsonValues       `json:",omitempty"`
	PathPatternConfig       *jsonValues       `json:",omitempty"`
	HttpRequestMethodConfig *jsonValues       `json:",omitempty"`
	SourceIpConfig          *jsonValues       `json:",omitempty"`
	HttpHeaderConfig        *httpHeaderValues `json:",omitempty"`
}

type jsonValues struct {
	Values json.RawMessage
}

type httpHeaderValues struct {
	jsonValues
	HttpHeaderName string
}

type LoadBalancerProperties struct {
	Name                   string
	Type                   string
	Scheme                 string
	IpAddressType          string
	SecurityGroups         []string
	Subnets                []string
	Tags                   []keyvaluePair
	LoadBalancerAttributes []keyvaluePair
}

type ListenerProperties struct {
	LoadBalancerArn string
	Certificates    []CertificateProperty
	DefaultActions  []ActionProperty
	Port            int
	Protocol        string
}

type ActionProperty struct {
	//AuthenticateCognitoConfig *AuthenticateCognitoActionConfig
	//FixedResponseConfig       *FixedResponseActionConfig
	//ForwardConfig             *ForwardActionConfig
	//RedirectConfig            *RedirectActionConfig
	Type           string
	Order          int           `json:",omitempty"`
	Oidc           *albEventOidc `json:"AuthenticateOidcConfig,omitempty"`
	TargetGroupArn *stringOrRef  `json:",omitempty"`
}

type CertificateProperty struct {
}

func calculatePriority(conds []cfnCondition) int {
	/*
		Rule limits for condition values, wildcards, and total rules.
		100 total rules per application load balancer
		5 condition values per rule
		5 wildcards per rule
		5 weighted target groups per rule

		host
		ip
		path
		method
		header
	*/

	crc32q := crc32.MakeTable(0xD5828281)
	cksum := func(val json.RawMessage, ceil int) int {
		sum32 := crc32.Checksum(val, crc32q)
		return int(sum32) % ceil
	}

	priority := 49_999

	for _, cond := range conds {
		if cond.HostHeaderConfig != nil && len(cond.HostHeaderConfig.Values) > 0 {
			priority -= cksum(cond.HostHeaderConfig.Values, 49_000)
		}

		if cond.SourceIpConfig != nil && len(cond.SourceIpConfig.Values) > 0 {
			priority -= cksum(cond.SourceIpConfig.Values, 49_000)
		}

		if cond.PathPatternConfig != nil && len(cond.PathPatternConfig.Values) > 0 {
			priority -= cksum(cond.PathPatternConfig.Values, 1_000)
		}

		if cond.HttpRequestMethodConfig != nil && len(cond.HttpRequestMethodConfig.Values) > 0 {
			priority -= cksum(cond.HttpRequestMethodConfig.Values, 100)
		}

		panic("IMPLEMENT ME")
		//if cond.HttpHeaderConfig != nil && len(cond.HttpHeaderConfig.Values) > 0 {
		//	priority -= cksum(cond.HttpHeaderConfig.Values, 10)
		//}
	}

	return priority
}

func convertConditions(input albEventConditions) []cfnCondition {
	output := []cfnCondition{}

	if len(input.Host) > 0 {
		output = append(output, cfnCondition{
			Field:            "host-header",
			HostHeaderConfig: &jsonValues{Values: input.Host},
		})
	}

	if len(input.Path) > 0 {
		output = append(output, cfnCondition{
			Field:             "path-pattern",
			PathPatternConfig: &jsonValues{Values: input.Path},
		})
	}

	if len(input.Method) > 0 {
		output = append(output, cfnCondition{
			Field:                   "http-request-method",
			HttpRequestMethodConfig: &jsonValues{Values: input.Method},
		})
	}

	for name, values := range input.Header {
		output = append(output, cfnCondition{
			Field: "http-header",
			HttpHeaderConfig: &httpHeaderValues{
				HttpHeaderName: name,
				jsonValues:     jsonValues{Values: values},
			},
		})
	}

	if len(input.Ip) > 0 {
		output = append(output, cfnCondition{
			Field:          "source-ip",
			SourceIpConfig: &jsonValues{Values: input.Ip},
		})
	}

	return output
}

func loadBalancerJson(props LoadBalancerProperties) []byte {
	bytes, _ := json.Marshal(props)
	return []byte(fmt.Sprintf(`
		{
			"Type": "AWS::ElasticLoadBalancingV2::LoadBalancer",
			"Properties": %s
		}
	`, string(bytes)))
}

func trailingTagsJson(tags map[string]json.RawMessage) string {
	if len(tags) == 0 {
		return ""
	}

	keys := []string{}
	for k, _ := range tags {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	buf := &bytes.Buffer{}
	buf.WriteString(`,
			"Tags": [`)

	for _, k := range keys {
		buf.WriteString(fmt.Sprintf(`{"Key": "%s", "Value": %s},`, k, string(tags[k])))
	}

	buf.Truncate(buf.Len() - 1) // chomp comma
	buf.WriteString("]")

	return buf.String()
}

func targetGroupJson(functionName, targetName string, tags map[string]json.RawMessage) []byte {
	return []byte(fmt.Sprintf(`
		{
			"DependsOn": ["%sAlbPermission"],
			"Type": "AWS::ElasticLoadBalancingV2::TargetGroup",
			"Properties": {
				"TargetType": "lambda",
				"Targets": [
					{"Id": {"Fn::GetAtt": ["%s", "Arn"]}}
				]%s
			}
		}
`, functionName, targetName, trailingTagsJson(tags)))
}

func permissionJson(targetName string) []byte {
	return []byte(fmt.Sprintf(`
		{
			"Type": "AWS::Lambda::Permission",
			"Properties": {
				"Action": "lambda:InvokeFunction",
				"Principal": "elasticloadbalancing.amazonaws.com",
				"SourceArn": {"Fn::Sub": "arn:aws:elasticloadbalancing:${AWS::Region}:${AWS::AccountId}:targetgroup/*"},
				"FunctionName": {"Fn::GetAtt": ["%s", "Arn"]}
			}
		}
	`, targetName))
}

func httpsListenerJson() []byte {
	return []byte(``)
}

func httpListenerJson() []byte {
	return []byte(``)
}

func listenerRuleJson(rule rule) []byte {
	bytes, _ := json.Marshal(rule)
	return []byte(fmt.Sprintf(`
		{
			"Type": "AWS::ElasticLoadBalancingV2::ListenerRule",
			"Properties": %s
		}
	`, string(bytes)))
}
