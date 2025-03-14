package state

import (
	"testing"

	"github.com/kong/go-kong/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var presetLicense = License{
	License: kong.License{
		ID:      kong.String("license-preset"),
		Payload: kong.String("preset-license-payload"),
	},
}

func TestLicenseCollection_Add(t *testing.T) {
	testCases := []struct {
		name          string
		license       *License
		expectedError error
	}{
		{
			name: "insert with no ID",
			license: &License{
				License: kong.License{},
			},
			expectedError: errIDRequired,
		},
		{
			name: "insert with ID and payload",
			license: &License{
				License: kong.License{
					ID:      kong.String("1234"),
					Payload: kong.String("license-test"),
				},
			},
		},
		{
			name: "insert a license with existing ID",
			license: &License{
				License: kong.License{
					ID:      kong.String("license-preset"),
					Payload: kong.String("license-test"),
				},
			},
			expectedError: ErrAlreadyExists,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			initialState := state()
			c := initialState.Licenses
			err := c.Add(presetLicense)
			require.NoError(t, err)

			err = c.Add(*tc.license)
			if tc.expectedError != nil {
				assert.ErrorIs(t, err, tc.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestLicenseCollection_Get(t *testing.T) {
	testCases := []struct {
		name            string
		id              string
		expectedPayload string
		expectedError   error
	}{
		{
			name:            "get existing license",
			id:              "license-preset",
			expectedPayload: "preset-license-payload",
		},
		{
			name:          "get non existing license",
			id:            "license-non-exist",
			expectedError: ErrNotFound,
		},
		{
			name:          "get with empty ID",
			id:            "",
			expectedError: errIDRequired,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			initialState := state()
			c := initialState.Licenses
			err := c.Add(presetLicense)
			require.NoError(t, err)

			l, err := c.Get(tc.id)
			if tc.expectedError == nil {
				require.NoError(t, err)
				assert.Equal(t, tc.id, *l.ID)
				assert.Equal(t, tc.expectedPayload, *l.Payload)
			} else {
				assert.ErrorIs(t, err, tc.expectedError)
			}
		})
	}
}

func TestLicenseCollection_Update(t *testing.T) {
	testCases := []struct {
		name          string
		license       License
		expectedError error
	}{
		{
			name:          "update with no ID",
			license:       License{},
			expectedError: errIDRequired,
		},
		{
			name: "update non existing license",
			license: License{
				License: kong.License{
					ID:      kong.String("license-non-exist"),
					Payload: kong.String("updated-payload"),
				},
			},
			expectedError: ErrNotFound,
		},
		{
			name: "update existing license",
			license: License{
				License: kong.License{
					ID:      kong.String("license-preset"),
					Payload: kong.String("updated-payload"),
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			initialState := state()
			c := initialState.Licenses
			err := c.Add(presetLicense)
			require.NoError(t, err)

			err = c.Update(tc.license)
			if tc.expectedError == nil {
				require.NoError(t, err)
				updatedLicense, err := c.Get(*tc.license.ID)
				require.NoError(t, err)
				assert.Equal(t, *tc.license.Payload, *updatedLicense.Payload)
			} else {
				assert.ErrorIs(t, err, tc.expectedError)
			}
		})
	}
}

func TestLicenseCollection_Delete(t *testing.T) {
	testCases := []struct {
		name          string
		id            string
		expectedError error
	}{
		{
			name:          "delete with no ID",
			id:            "",
			expectedError: errIDRequired,
		},
		{
			name:          "delete non existing license",
			id:            "license-non-exist",
			expectedError: ErrNotFound,
		},
		{
			name: "delete existing license",
			id:   "license-preset",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			initialState := state()
			c := initialState.Licenses
			err := c.Add(presetLicense)
			require.NoError(t, err)

			err = c.Delete(tc.id)
			if tc.expectedError == nil {
				require.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, tc.expectedError)
			}
		})
	}
}

func TestLicenseCollection_GetAll(t *testing.T) {
	initialState := state()
	c := initialState.Licenses
	licenses, err := c.GetAll()
	require.NoError(t, err)
	assert.Empty(t, licenses, "Should have no licenses")

	err = c.Add(presetLicense)
	require.NoError(t, err)

	licenses, err = c.GetAll()
	require.NoError(t, err)
	assert.Len(t, licenses, 1, "Should have 1 license after adding")
}
