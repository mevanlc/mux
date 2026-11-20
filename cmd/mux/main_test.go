package main

import (
	"reflect"
	"strings"
	"testing"

	"github.com/mevanlc/mux/tmux"
	"github.com/spf13/cobra"
)

func TestSocketFlag(t *testing.T) {
	defer tmux.SetSocket("")
	for _, args := range [][]string{
		{"-L", "work"},
		{"--socket", "work"},
		{"--socket=work"},
		{"-L", "work", "popup"},
		{"popup", "--socket", "work"},
		{"status", "-L", "work"},
		{"setup-keybind", "--socket", "work"},
		{},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			root := newRootCommand()
			var got []string
			run := func(cmd *cobra.Command, args []string) error {
				got = tmux.SocketArgs("list-sessions")
				return nil
			}
			root.RunE = run
			for _, cmd := range root.Commands() {
				cmd.RunE = run
			}
			root.SetArgs(args)
			if err := root.Execute(); err != nil {
				t.Fatal(err)
			}
			want := []string{"-L", "work", "list-sessions"}
			if len(args) == 0 {
				want = []string{"list-sessions"}
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("tmux args = %q, want %q", got, want)
			}
		})
	}
}
