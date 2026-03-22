package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/shenghuikevin/cc-track/internal/agent"
	"github.com/shenghuikevin/cc-track/internal/analysis"
	"github.com/shenghuikevin/cc-track/internal/config"
	"github.com/shenghuikevin/cc-track/internal/output"
	"github.com/shenghuikevin/cc-track/internal/store"
	"github.com/spf13/cobra"
)

var (
	wasteSessionID string
	wasteAgent     bool
	wasteProvider  string
	wasteModel     string
)

var wasteCmd = &cobra.Command{
	Use:   "waste",
	Short: "Detect wasteful patterns in sessions",
	Long: `Detect wasteful patterns in sessions.

Use --agent to get AI-powered improvement suggestions.

Providers: zhipu (default), minimax
Set API key via environment variable: CC_TRACK_API_KEY

Examples:
  cc-track waste                          # local rules only
  cc-track waste --agent                  # with AI suggestions (zhipu GLM-5)
  cc-track waste --agent --provider minimax
  cc-track waste --agent --model GLM-4.7  # override model`,
	RunE: runWaste,
}

func init() {
	wasteCmd.Flags().StringVar(&wasteSessionID, "session", "", "analyze a specific session (ID or prefix)")
	wasteCmd.Flags().BoolVar(&wasteAgent, "agent", false, "use AI agent for improvement suggestions")
	wasteCmd.Flags().StringVar(&wasteProvider, "provider", agent.DefaultProvider, "AI provider: zhipu, minimax")
	wasteCmd.Flags().StringVar(&wasteModel, "model", "", "override model name")
	rootCmd.AddCommand(wasteCmd)
}

func runWaste(cmd *cobra.Command, args []string) error {
	dbPath, err := config.DBPath()
	if err != nil {
		return err
	}
	s, err := store.Open(dbPath)
	if err != nil {
		return err
	}
	defer s.Close()

	var sessionIDs []string
	if wasteSessionID != "" {
		sess, err := s.FindSessionByPrefix(wasteSessionID)
		if err != nil {
			return err
		}
		sessionIDs = []string{sess.ID}
	} else {
		ids, err := s.GetRecentSessionIDs(10)
		if err != nil {
			return err
		}
		sessionIDs = ids
	}

	if len(sessionIDs) == 0 {
		fmt.Println("No sessions found.")
		return nil
	}

	report, err := analysis.AnalyzeWaste(s, sessionIDs)
	if err != nil {
		return err
	}

	// Agent mode: get AI suggestions
	var suggestion string
	if wasteAgent {
		client, err := buildAgentClient()
		if err != nil {
			return err
		}
		if jsonOutput {
			j, err := agent.SuggestJSON(client, report)
			if err != nil {
				return fmt.Errorf("agent suggestion failed: %w", err)
			}
			fmt.Println(j)
			return nil
		}
		suggestion, err = agent.Suggest(client, report)
		if err != nil {
			return fmt.Errorf("agent suggestion failed: %w", err)
		}
	}

	if jsonOutput {
		j, err := output.JSON(report)
		if err != nil {
			return err
		}
		fmt.Println(j)
		return nil
	}

	printWasteReport(report)
	if suggestion != "" {
		fmt.Printf("\n─── AI Suggestions (%s/%s) ───\n\n", wasteProvider, getModelName())
		fmt.Println(suggestion)
	}
	return nil
}

func buildAgentClient() (*agent.Client, error) {
	provider, ok := agent.GetProvider(wasteProvider)
	if !ok {
		available := strings.Join(agent.ListProviders(), ", ")
		return nil, fmt.Errorf("unknown provider %q (available: %s)", wasteProvider, available)
	}

	// Load API key from env
	apiKey := os.Getenv("CC_TRACK_API_KEY")
	if apiKey == "" {
		// Try provider-specific env vars
		switch wasteProvider {
		case "zhipu":
			apiKey = os.Getenv("CC_TRACK_ZHIPU_API_KEY")
		case "minimax":
			apiKey = os.Getenv("CC_TRACK_MINIMAX_API_KEY")
		}
	}
	if apiKey == "" {
		return nil, fmt.Errorf("API key not set. Set CC_TRACK_API_KEY or CC_TRACK_%s_API_KEY environment variable",
			strings.ToUpper(wasteProvider))
	}
	provider.APIKey = apiKey

	// Override model if specified
	if wasteModel != "" {
		provider.Model = wasteModel
	}

	return agent.NewClient(provider), nil
}

func getModelName() string {
	if wasteModel != "" {
		return wasteModel
	}
	p, ok := agent.GetProvider(wasteProvider)
	if ok {
		return p.Model
	}
	return "unknown"
}

func printWasteReport(report *analysis.WasteReport) {
	fmt.Printf("Waste Analysis (%d sessions analyzed)\n\n", report.SessionsAnalyzed)

	if len(report.Findings) == 0 {
		fmt.Println("  No waste patterns detected. Nice!")
		return
	}

	fmt.Printf("  Found %d issue(s):\n\n", len(report.Findings))

	typeLabels := map[analysis.WasteType]string{
		analysis.WasteDuplicateCalls: "Duplicate Calls",
		analysis.WasteExcessiveReads: "Excessive Reads",
		analysis.WasteFailedRetries:  "Failed Retries",
		analysis.WasteEditRevert:     "Edit Revert",
		analysis.WasteZombieSession:  "Zombie Session",
	}

	for i, f := range report.Findings {
		label := typeLabels[f.Type]
		if label == "" {
			label = string(f.Type)
		}
		fmt.Printf("  %d. [%s] %s\n", i+1, label, f.Summary)
		if f.SessionID != "" {
			fmt.Printf("     Session: %s\n", f.SessionID[:min(12, len(f.SessionID))])
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
