package cmd

import (
	"fmt"
	"log"
	"time"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/spf13/cobra"
)

func scheduleCmd() *cobra.Command {
	cmd := cobra.Command{
		Use: "schedule",
	}

	cmd.AddCommand(listSchedulesCmd())
	cmd.AddCommand(showScheduleCmd())

	return &cmd
}

func listSchedulesCmd() *cobra.Command {
	cmd := cobra.Command{
		Use: "list",
		Run: listSchedulesAction,
	}

	return &cmd
}

func listSchedulesAction(cmd *cobra.Command, args []string) {
	pd := client()

	u, err := pd.GetCurrentUser(pagerduty.GetCurrentUserOptions{})
	if err != nil {
		log.Fatalf("get current user err: %s", err)
	}

	var offset int
	opts := pagerduty.ListSchedulesOptions{}
	for more := true; more; {
		opts.APIListObject = pagerduty.APIListObject{
			Limit:  100,
			Offset: uint(offset),
		}

		lsr, err := pd.ListSchedules(opts)
		if err != nil {
			log.Fatalf("List Schedules err: %s", err)
		}

		more = lsr.More
		offset += len(lsr.Schedules)
		for _, sched := range lsr.Schedules {
			for _, user := range sched.Users {
				if user.ID == u.ID {
					fmt.Printf("%s %s\n", sched.ID, sched.Name)
				}
			}
		}
	}
}

func showScheduleCmd() *cobra.Command {
	cmd := cobra.Command{
		Use: "show <id>",
		Run: showScheduleAction,
	}

	return &cmd
}

func showScheduleAction(cmd *cobra.Command, args []string) {
	pd := client()

	if len(args) < 1 {
		log.Fatalf("usage: show <schedule_id>")
	}

	start := time.Now().Add(-24 * time.Hour).Format(time.RFC3339)
	end := time.Now().Add(7 * 24 * time.Hour).Format(time.RFC3339)

	sched, err := pd.GetSchedule(args[0], pagerduty.GetScheduleOptions{
		TimeZone: "America/Los_Angeles",
		Since:    start,
		Until:    end,
	})
	if err != nil {
		log.Fatalf("get schedule err: %s", err)
	}

	for _, entry := range sched.FinalSchedule.RenderedScheduleEntries {
		fmt.Printf("%s %s %s\n", entry.Start, entry.End, entry.User.Summary)
	}
}
