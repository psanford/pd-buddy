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
	cmd.AddCommand(ackIncidentCmd())
	cmd.AddCommand(resolveIncidentCmd())

	return &cmd
}

var (
	scopeFlag  string
	statusFlag string

	allowTeamAckFlag     bool
	allowAllTeamsAckFlag bool
	skipConfirmFlag      bool
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

func ackIncidentCmd() *cobra.Command {
	cmd := cobra.Command{
		Use: "ack",
		Run: ackIncidentAction,
	}

	cmd.Flags().BoolVarP(&allowTeamAckFlag, "allow-team", "", false, "Allow acking incidents for my team not assigned to me")
	cmd.Flags().BoolVarP(&allowAllTeamsAckFlag, "allow-all-teams-i-know-this-is-dangerous", "", false, "Allow acking incidents for other teams (dangerous!)")
	cmd.Flags().BoolVarP(&skipConfirmFlag, "--yes", "", false, "Don't prompt for confirmation before acking")

	return &cmd
}

func ackIncidentAction(cmd *cobra.Command, args []string) {
	if len(args) < 1 {
		log.Fatalf("usage: ack <incident_number> [...<incident_number>]")
	}

	ackOrResolveIncidents("acknowledged", args)
}

func resolveIncidentCmd() *cobra.Command {
	cmd := cobra.Command{
		Use: "resolve",
		Run: resolveIncidentAction,
	}

	cmd.Flags().BoolVarP(&allowTeamAckFlag, "allow-team", "", false, "Allow acking incidents for my team not assigned to me")
	cmd.Flags().BoolVarP(&allowAllTeamsAckFlag, "allow-all-teams-i-know-this-is-dangerous", "", false, "Allow acking incidents for other teams (dangerous!)")
	cmd.Flags().BoolVarP(&skipConfirmFlag, "--yes", "", false, "Don't prompt for confirmation before acking")

	return &cmd
}

func resolveIncidentAction(cmd *cobra.Command, args []string) {
	if len(args) < 1 {
		log.Fatalf("usage: resolve <incident_number> [...<incident_number>]")
	}

	ackOrResolveIncidents("resolved", args)
}

func ackOrResolveIncidents(action string, incidentIDs []string) {
	pd := client()

	u, err := pd.GetCurrentUser(pagerduty.GetCurrentUserOptions{})
	if err != nil {
		log.Fatalf("get current user err: %s", err)
	}

	myTeams := make(map[string]string)
	for _, t := range u.Teams {
		myTeams[t.ID] = t.Name
	}

	for _, incidentID := range incidentIDs {
		incd, err := pd.GetIncident(incidentID)
		if err != nil {
			log.Fatalf("fetch incident %s err: %s", incidentID, err)
		}

		var (
			assigedToMe     bool
			assigedToMyTeam bool
			allowedToUpdate bool
		)

		for _, assignment := range incd.Assignments {
			if assignment.Assignee.ID == u.ID {
				assigedToMe = true
			}
		}

		for _, team := range incd.Teams {
			if _, exists := myTeams[team.ID]; exists {
				assigedToMyTeam = true
			}
		}

		if assigedToMe {
			allowedToUpdate = true
		}

		if assigedToMyTeam && allowTeamAckFlag {
			allowedToUpdate = true
		}

		if allowAllTeamsAckFlag {
			allowedToUpdate = true
		}

		if !allowedToUpdate {
			if !assigedToMyTeam {
				log.Fatalf("incd %s %s is not assigned to me or my team", incidentID, incd.Summary)
			}
			if assigedToMyTeam && !allowTeamAckFlag {
				log.Fatalf("incd %s %s is not assigned to my team but no --allow-team flag specified", incidentID, incd.Summary)
			}

			log.Fatalf("Not allowed to update for unkown reason, this is a bug")
		}

		if !skipConfirmFlag {
			var assignedTo string
			if len(incd.Assignments) > 0 {
				assignedTo = incd.Assignments[0].Assignee.Summary
			}
			fmt.Printf("%s %d %s %s %s\n", incd.CreatedAt, incd.IncidentNumber, incd.Status, assignedTo, incd.Description)
			fmt.Printf("\n%s => %s\n\n", incd.Status, action)
			ok := confirm("Update incident status [yN]? ")
			if !ok {
				log.Fatal("Aborting")
			}
		}

		_, err = pd.ManageIncidents(u.Email, []pagerduty.ManageIncidentsOptions{
			{
				ID:     incd.ID,
				Status: action,
			},
		})

		if err != nil {
			log.Fatalf("Update incident %s err: %s %+v", incidentID, err, err)
		}
	}
}
