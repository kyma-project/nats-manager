package file

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_DirExists(t *testing.T) {
	t.Parallel()

	t.Run("Test chart configuration processing", func(t *testing.T) {
		t.Parallel()
		// define test cases
		testCases := []struct {
			name       string
			path       string
			wantResult bool
		}{
			{
				name:       "should return true for existing directory",
				path:       "../file",
				wantResult: true,
			},
			{
				name:       "should return false for non-existing directory",
				path:       "./not_exists123",
				wantResult: false,
			},
			{
				name:       "should return false if path is empty",
				path:       "",
				wantResult: false,
			},
		}

		// run test cases
		for _, tc := range testCases {
			tc := tc
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				require.Equal(t, tc.wantResult, DirExists(tc.path))
			})
		}
	})
}
