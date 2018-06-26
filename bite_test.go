package bite

import (
	"fmt"
	"testing"

	"github.com/spf13/cobra"
)

func TestGetNested(t *testing.T) {
	var (
		expectedName      = "my-app"
		expectedDesc      = "my-description"
		bottomCommandName = "my-my-my-command"
	)

	app := Name(expectedName).Description(expectedDesc).Get()

	level1 := &cobra.Command{Use: "my-command"}
	level2 := &cobra.Command{Use: "my-my-command"}
	level22 := &cobra.Command{Use: "my-my-second-command"}
	level3 := &cobra.Command{Use: bottomCommandName, RunE: func(cmd *cobra.Command, args []string) error {
		got := Get(cmd)
		if got == nil {
			return fmt.Errorf("app can not be found from the command: %s", cmd.Name())
		}
		if got.Name != expectedName {
			return fmt.Errorf("expected application name to be '%s' but got '%s'", expectedName, got.Name)
		}

		if got.Description != expectedDesc {
			return fmt.Errorf("expected application description to be '%s' but got '%s'", expectedDesc, got.Description)
		}

		return fmt.Errorf("command ran")
	}}

	level2.AddCommand(level3)
	level1.AddCommand(level2)
	level1.AddCommand(level22)
	app.AddCommand(level1)
	Build(app)

	c := app.GetCommand(bottomCommandName)
	// or c := GetCommand(app.Name, bottomCommandName)
	// or c, _ := app.FindCommand([]string{"my-command", "my-my-command", bottomCommandName})
	if c == nil {
		t.Fatalf("unknown command '%s' for '%s'", app.Name, bottomCommandName)
	}

	err := c.RunE(c, nil)

	// we expect an error, in order to test if command ran at least.
	if err == nil {
		t.Fatalf("expected to run the latest command but nothing ran")
	}

	if err.Error() != "command ran" {
		t.Fatal(err)
	}
}
