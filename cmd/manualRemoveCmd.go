package cmd

import (
	"bufio"
	"fmt"
	"github.com/afc98/aqualog/core/manual"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var manualRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove a manual water level reading by its ID",
	Run: func(cmd *cobra.Command, args []string) {
		id, _ := cmd.Flags().GetInt("id")

		// Remove manual reading
		existing, err := manual.Get(id)
		if err != nil {
			fmt.Println("Error fetching manual reading:", err)
			return
		}

		// Show details
		fmt.Printf("Are you sure you want to delete manual reading %d?\n", id)
		fmt.Printf(
			"%d. Site %d (%s) Water Level: %.2f Notes: %s\n",
			existing.ID,
			existing.SiteID,
			existing.Timestamp.Format("2006-01-02 15:04:05"),
			existing.WaterLevel,
			existing.Notes,
		)

		fmt.Print("Enter Y to confirm: ")

		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		if strings.TrimSpace(input) != "Y" {
			fmt.Println("Cancelled.")
			return
		}

		if err := manual.Delete(id); err != nil {
			fmt.Println("Error removing manual reading:", err)
			return
		}

		fmt.Println("Manual reading removed.")

	},
}

func init() {
	manualRemoveCmd.Flags().IntP("id", "i", 0, "Manual reading ID to remove")
	manualRemoveCmd.MarkFlagRequired("id")

	manualCmd.AddCommand(manualRemoveCmd)
}
