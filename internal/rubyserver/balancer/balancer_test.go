package balancer

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRemovals(t *testing.T) {
	okActions := []action{
		{add: "foo"},
		{add: "bar"},
		{add: "qux"},
		{remove: "bar"},
		{remove: "foo"},
	}

	testCases := []struct {
		desc      string
		actions   []action
		lastFails bool
	}{
		{
			desc:    "add then remove",
			actions: okActions,
		},
		{
			desc:      "remove last address",
			actions:   append(okActions, action{remove: "qux"}),
			lastFails: true,
		},
		{
			desc: "remove unknown address",
			actions: []action{
				{add: "foo"},
				{remove: "bar"},
			},
			lastFails: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			// This breaks integration with gRPC and causes a monitor goroutine leak.
			// Not a problem for this test.
			lbBuilder = newBuilder()

			for i, a := range tc.actions {
				if a.add != "" {
					AddAddress(a.add)
				} else {
					expected := true
					if i+1 == len(tc.actions) && tc.lastFails {
						expected = false
					}

					require.Equal(t, expected, RemoveAddress(a.remove), "expected result from removing %q", a.remove)
				}
			}
		})
	}
}

type action struct {
	add    string
	remove string
}
