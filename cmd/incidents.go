package cmd

import (
	"fmt"
	"log"
	"strings"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/spf13/cobra"
)

func incidentCmd() *cobra.Command {
	cmd := cobra.Command{
		Use: "incident",
	}

	cmd.AddCommand(listIncidentsCmd())

	return &cmd
}

var (
	scopeFlag  string
	statusFlag string
)

func listIncidentsCmd() *cobra.Command {
	cmd := cobra.Command{
		Use: "list",
		Run: listIncidentsAction,
	}

	cmd.Flags().StringVarP(&scopeFlag, "scope", "", "me", "Limit to my incidents or team's incidents (me|team)")
	cmd.Flags().StringVarP(&statusFlag, "status", "", "triggered,acknowledged", "Comma seperated list of status to limit to (triggered,acknowledged,resolved)")

	return &cmd
}

func listIncidentsAction(cmd *cobra.Command, args []string) {
	pd := client()

	u, err := pd.GetCurrentUser(pagerduty.GetCurrentUserOptions{})
	if err != nil {
		log.Fatalf("get current user err: %s", err)
	}

	opts := pagerduty.ListIncidentsOptions{}

	if scopeFlag == "me" {
		opts.UserIDs = []string{u.ID}
	} else if scopeFlag == "team" {
		for _, t := range u.Teams {
			opts.TeamIDs = append(opts.TeamIDs, t.ID)
		}
	} else {
		log.Fatalf("invalid option for -scope, must be me|team")
	}

	if statusFlag != "" {
		opts.Statuses = strings.Split(statusFlag, ",")
	}

	var offset int
	for more := true; more; {
		opts.APIListObject = pagerduty.APIListObject{
			Limit:  100,
			Offset: uint(offset),
		}

		incdresp, err := pd.ListIncidents(opts)
		if err != nil {
			log.Fatalf("list incidents err: %s", err)
		}

		more = incdresp.More
		offset += len(incdresp.Incidents)
		for _, incd := range incdresp.Incidents {
			var assignedTo string
			if len(incd.Assignments) > 0 {
				assignedTo = incd.Assignments[0].Assignee.Summary
			}
			fmt.Printf("%s %d %s %s %s\n", incd.CreatedAt, incd.IncidentNumber, incd.Status, assignedTo, incd.Description)
		}
	}

}
