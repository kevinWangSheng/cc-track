package cmd

import (
	"fmt"
	"time"

	"github.com/shenghuikevin/cc-track/internal/config"
	"github.com/shenghuikevin/cc-track/internal/output"
	"github.com/shenghuikevin/cc-track/internal/store"
	"github.com/spf13/cobra"
)

var sessionLimit int

var sessionCmd = &cobra.Command{
	Use:   "session",
	Short: "Session management",
}

var sessionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List recent sessions",
	RunE:  runSessionList,
}

var sessionShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show session details and timeline",
	Args:  cobra.ExactArgs(1),
	RunE:  runSessionShow,
}

func init() {
	sessionListCmd.Flags().IntVar(&sessionLimit, "limit", 10, "max sessions to show")
	sessionCmd.AddCommand(sessionListCmd)
	sessionCmd.AddCommand(sessionShowCmd)
	rootCmd.AddCommand(sessionCmd)
}

func openStore() (*store.Store, error) {
	dbPath, err := config.DBPath()
	if err != nil {
		return nil, err
	}
	return store.Open(dbPath)
}

func runSessionList(cmd *cobra.Command, args []string) error {
	s, err := openStore()
	if err != nil {
		return err
	}
	defer s.Close()

	sessions, err := s.ListSessions(sessionLimit)
	if err != nil {
		return err
	}

	if jsonOutput {
		j, err := output.JSON(sessions)
		if err != nil {
			return err
		}
		fmt.Println(j)
		return nil
	}

	if len(sessions) == 0 {
		fmt.Println("no sessions found")
		return nil
	}

	t := output.NewTable("ID", "Project", "Branch", "Model", "Started", "Prompts", "Tools")
	for _, sess := range sessions {
		started := time.UnixMilli(sess.StartedAt).Format("01-02 15:04")
		id := sess.ID
		if len(id) > 12 {
			id = id[:12]
		}
		t.AddRow(
			id,
			sess.ProjectStr,
			sess.BranchStr,
			sess.ModelStr,
			started,
			fmt.Sprintf("%d", sess.TotalPrompts),
			fmt.Sprintf("%d", sess.TotalToolCalls),
		)
	}
	fmt.Print(t.String())
	return nil
}

func runSessionShow(cmd *cobra.Command, args []string) error {
	s, err := openStore()
	if err != nil {
		return err
	}
	defer s.Close()

	tl, err := s.GetSessionTimeline(args[0])
	if err != nil {
		return err
	}

	if jsonOutput {
		j, err := output.JSON(tl)
		if err != nil {
			return err
		}
		fmt.Println(j)
		return nil
	}

	printTimeline(tl)
	return nil
}

func printTimeline(tl *store.SessionTimeline) {
	sess := tl.Session
	fmt.Printf("Session: %s\n", sess.ID)
	fmt.Printf("  Project:  %s\n", sess.ProjectStr)
	fmt.Printf("  Branch:   %s\n", sess.BranchStr)
	fmt.Printf("  Model:    %s\n", sess.ModelStr)
	fmt.Printf("  CWD:      %s\n", sess.CWD)
	fmt.Printf("  Started:  %s\n", time.UnixMilli(sess.StartedAt).Format(time.DateTime))
	if sess.EndedAtVal > 0 {
		fmt.Printf("  Ended:    %s\n", time.UnixMilli(sess.EndedAtVal).Format(time.DateTime))
		dur := time.Duration(sess.EndedAtVal-sess.StartedAt) * time.Millisecond
		fmt.Printf("  Duration: %s\n", formatDuration(int64(dur/time.Millisecond)))
	}
	fmt.Printf("  Prompts:  %d  Tools: %d\n", sess.TotalPrompts, sess.TotalToolCalls)
	if sess.TotalTokens() > 0 {
		fmt.Printf("  Tokens:   %s (in: %s, out: %s, cache-read: %s, cache-create: %s)\n",
			formatTokens(sess.TotalTokens()),
			formatTokens(sess.TotalInputTokens),
			formatTokens(sess.TotalOutputTokens),
			formatTokens(sess.TotalCacheReadTokens),
			formatTokens(sess.TotalCacheCreationTokens),
		)
	}

	fmt.Println("\nTimeline:")
	for _, p := range tl.Prompts {
		ts := time.UnixMilli(p.Timestamp).Format("15:04:05")
		text := p.PromptText
		if len(text) > 80 {
			text = text[:80] + "..."
		}
		fmt.Printf("  %s  [prompt] %s\n", ts, text)
	}
	for _, tc := range tl.ToolCalls {
		ts := time.UnixMilli(tc.StartedAt).Format("15:04:05")
		status := "ok"
		if tc.Succeeded == 0 {
			status = "FAIL"
		}
		durStr := ""
		if tc.DurationMsVal > 0 {
			durStr = fmt.Sprintf(" (%dms)", tc.DurationMsVal)
		}
		fmt.Printf("  %s  [tool]   %-20s %s%s\n", ts, tc.ToolName, status, durStr)
	}
	for _, se := range tl.StopEvents {
		ts := time.UnixMilli(se.Timestamp).Format("15:04:05")
		detail := ""
		if se.ErrorType != "" {
			detail = " - " + se.ErrorType
		}
		fmt.Printf("  %s  [stop]   %s%s\n", ts, se.EventType, detail)
	}
}
