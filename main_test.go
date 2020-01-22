package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"
)

func TestMyvisitor_VisitResource(t *testing.T) {
	fragment, err := ioutil.ReadFile("testdata/no_alb_event.json")
	assert.NoError(t, err)

	expected := make([]byte, len(fragment))
	copy(expected, fragment)

	v := &myvisitor{}
	tf := &templateFragment{fragment: fragment}

	err = tf.Visit(context.Background(), v)
	assert.NoError(t, err)
	assert.Empty(t, v.props)
	assert.Equal(t, expected, []byte(tf.fragment))
}

type mockVisitor struct {
	mock.Mock
}

func (m *mockVisitor) VisitResource(ctx context.Context, t *templateFragment, name, resourceType string, resource json.RawMessage) error {
	f := m.Called(ctx, t, name, resourceType, resource)
	return f.Error(0)
}

func TestVisitation(t *testing.T) {
	fragment, err := ioutil.ReadFile("testdata/no_alb_event.json")
	assert.NoError(t, err)

	expected := make([]byte, len(fragment))
	copy(expected, fragment)

	tf := &templateFragment{fragment: fragment}

	m := &mockVisitor{}
	m.On("VisitResource", mock.Anything, tf, "Function", "AWS::Serverless::Function", mock.Anything).Return(nil).Once()
	m.On("VisitResource", mock.Anything, tf, "LogGroup", "AWS::Logs::LogGroup", mock.Anything).Return(nil).Once()

	err = tf.Visit(context.Background(), m)
	assert.NoError(t, err)
	assert.Equal(t, expected, []byte(tf.fragment))

	m.AssertExpectations(t)
}

func TestNonAlbEvent(t *testing.T) {
	fragment, err := ioutil.ReadFile("testdata/non_alb_event.json")
	assert.NoError(t, err)

	expected := make([]byte, len(fragment))
	copy(expected, fragment)

	v := &myvisitor{}
	tf := &templateFragment{fragment: fragment}

	err = tf.Visit(context.Background(), v)
	assert.NoError(t, err)
	assert.Empty(t, v.props)
	assert.Equal(t, expected, []byte(tf.fragment))
}

func TestEventWithoutProperties(t *testing.T) {
	fragment, err := ioutil.ReadFile("testdata/event_without_properties.json")
	assert.NoError(t, err)

	expected, err := ioutil.ReadFile("testdata/event_without_properties_result.json")
	assert.NoError(t, err)

	expectedProps := []albEventProperties{
		{Resource: "Function"},
	}

	v := &myvisitor{}
	tf := &templateFragment{fragment: fragment}

	err = tf.Visit(context.Background(), v)
	assert.NoError(t, err)
	assert.Equal(t, expectedProps, v.props)
	assert.JSONEq(t, string(expected), string(tf.fragment))
}

func TestTwoAlbEvents(t *testing.T) {
	fragment, err := ioutil.ReadFile("testdata/two_alb_events.json")
	assert.NoError(t, err)

	expected, err := ioutil.ReadFile("testdata/two_alb_events_result.json")
	assert.NoError(t, err)

	expectedProps := []albEventProperties{
		{Resource: "Function"},
		{Resource: "Function", Conditions: albEventConditions{Host: []string{"example.com"}}},
	}

	v := &myvisitor{}
	tf := &templateFragment{fragment: fragment}

	err = tf.Visit(context.Background(), v)
	assert.NoError(t, err)
	assert.Equal(t, expectedProps, v.props)
	assert.JSONEq(t, string(expected), string(tf.fragment))
}

func TestListenerIsRef(t *testing.T) {
	fragment, err := ioutil.ReadFile("testdata/listener_is_ref.json")
	assert.NoError(t, err)

	expected, err := ioutil.ReadFile("testdata/listener_is_ref_result.json")
	assert.NoError(t, err)

	input := &MacroInput{Fragment: fragment}
	output, err := New(nil).Handle(context.Background(), input)
	assert.NoError(t, err)
	assert.JSONEq(t, string(expected), string(output.Fragment))
}

func TestRefsEverywhere(t *testing.T) {
	fragment, err := ioutil.ReadFile("testdata/listener_is_ref.json")
	assert.NoError(t, err)

	expected, err := ioutil.ReadFile("testdata/listener_is_ref_result.json")
	assert.NoError(t, err)

	input := &MacroInput{Fragment: fragment}
	output, err := New(nil).Handle(context.Background(), input)
	assert.NoError(t, err)
	assert.JSONEq(t, string(expected), string(output.Fragment))
}

func TestVpcConfig(t *testing.T) {
	fragment, err := ioutil.ReadFile("testdata/vpc_config.json")
	assert.NoError(t, err)

	expected := []albEventProperties{
		{
			Resource: "Function",
			VpcConfig: &albEventVpcConfig{
				SubnetIds:        []string{"a", "b"},
				SecurityGroupIds: []string{"c", "d"},
			},
			Conditions: albEventConditions{
				Host: []string{"example.com"},
			},
		},
	}

	v := &myvisitor{}
	tf := &templateFragment{fragment: fragment}

	err = tf.Visit(context.Background(), v)
	assert.NoError(t, err)
	assert.Equal(t, expected, v.props)
}

func TestHandler_Handle(t *testing.T) {
	fragment, err := ioutil.ReadFile("testdata/single_alb_event.json")
	assert.NoError(t, err)

	expected, err := ioutil.ReadFile("testdata/single_alb_event_result.json")
	assert.NoError(t, err)

	input := &MacroInput{Fragment: fragment}
	output, err := New(nil).Handle(context.Background(), input)
	assert.NoError(t, err)
	assert.JSONEq(t, string(expected), string(output.Fragment))
}

func TestAllTheThings(t *testing.T) {
	paths, err := filepath.Glob("testdata/*_input.json")
	assert.NoError(t, err)

	handler := New(nil)

	for _, path := range paths {
		base := filepath.Base(path)
		t.Run(base, func(t *testing.T) {
			if strings.HasPrefix(base, "skip_") {
				t.SkipNow()
			}

			fragment, err := ioutil.ReadFile(path)
			assert.NoError(t, err)

			expected, err := ioutil.ReadFile(strings.Replace(path, "_input.json", "_result.json", 1))
			assert.NoError(t, err)

			input := &MacroInput{Fragment: fragment}
			output, err := handler.Handle(context.Background(), input)
			assert.NoError(t, err)
			require.NotNil(t, output)
			assert.JSONEq(t, string(expected), string(output.Fragment))
		})
	}

	fmt.Println(paths)
}
