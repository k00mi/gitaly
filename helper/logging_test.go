package helper

import (
	"testing"
)

func TestParsingCommandSuccessfully(t *testing.T) {
	cases := make(map[string][]string)
	cases["foo"] = []string{"--long-arg", "path/arg", "-f", "foo", "arg1", "arg2"}
	cases["bar"] = []string{"bar", "-d", "arg1", "arg2"}
	cases["baz"] = []string{"baz", "-f=\"some_value\""}

	for k, v := range cases {
		subCmd := ExtractSubcommand(v)
		if k != subCmd {
			t.Error("Expected subcommand for", v, "to be", k, ", got", subCmd)
		}
	}
}
