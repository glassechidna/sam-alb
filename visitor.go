package main

import (
	"context"
	"encoding/json"
	"github.com/buger/jsonparser"
	"github.com/pkg/errors"
)

type templateFragment struct {
	fragment json.RawMessage
}

type templateVisitor interface {
	VisitResource(ctx context.Context, t *templateFragment, name, resourceType string, resource json.RawMessage) error
}

type myvisitor struct {
	props []albEventProperties
}

func (tf *templateFragment) resourceNames() ([]string, error) {
	resources, _, _, err := jsonparser.Get(tf.fragment, "Resources")
	var resourceNames []string

	err = jsonparser.ObjectEach(resources, func(key []byte, value []byte, dataType jsonparser.ValueType, offset int) error {
		resourceNames = append(resourceNames, string(key))
		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "iterating over resources")
	}

	return resourceNames, nil
}

func (tf *templateFragment) PutResource(name string, resource json.RawMessage) error {
	f, err := jsonparser.Set(tf.fragment, resource, "Resources", name)
	if err != nil {
		return errors.WithStack(err)
	}

	tf.fragment = f
	return nil
}

func (tf *templateFragment) Visit(ctx context.Context, visitor templateVisitor) error {
	names, err := tf.resourceNames()
	if err != nil {
		return err
	}

	for _, name := range names {
		resourceBytes, _, _, err := jsonparser.Get(tf.fragment, "Resources", name)
		resType, err := jsonparser.GetString(resourceBytes, "Type")
		if err != nil {
			return errors.Wrap(err, "getting Resource type")
		}

		err = visitor.VisitResource(ctx, tf, name, resType, resourceBytes)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *myvisitor) VisitResource(ctx context.Context, t *templateFragment, name, resourceType string, resource json.RawMessage) error {
	if resourceType != "AWS::Serverless::Function" {
		return nil
	}

	var keysToDelete []string
	res := []byte(resource)

	alias, _ := jsonparser.GetString(resource, "Properties", "AutoPublishAlias")

	tagsjson, _, _, _ := jsonparser.Get(resource, "Properties", "Tags")
	tags := map[string]json.RawMessage{}
	if len(tagsjson) > 0 {
		json.Unmarshal(tagsjson, &tags)
	}

	err := jsonparser.ObjectEach(resource, func(key []byte, value []byte, dataType jsonparser.ValueType, offset int) error {
		eventType, err := jsonparser.GetString(value, "Type")
		if err != nil || eventType != "ALB" {
			return errors.WithStack(err)
		}

		propjson, _, _, err := jsonparser.Get(value, "Properties")
		prop := albEventProperties{}
		json.Unmarshal(propjson, &prop)
		prop.Resource = name
		prop.EventName = string(key)
		prop.Alias = alias
		prop.Tags = tags
		m.props = append(m.props, prop)

		keysToDelete = append(keysToDelete, string(key))
		return nil
	}, "Properties", "Events")

	if err != nil && err != jsonparser.KeyPathNotFoundError {
		return errors.WithStack(err)
	}

	for _, key := range keysToDelete {
		res = jsonparser.Delete(res, "Properties", "Events", key)
	}

	return t.PutResource(name, res)
}
