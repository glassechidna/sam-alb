package main

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"
)

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
