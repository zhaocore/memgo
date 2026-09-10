package cli

import "github.com/spf13/cobra"

// 共享 flag 注册 (选项名/短别名与 python CLI 逐一对应)。
type connFlags struct {
	apiKey  string
	baseURL string
}

func addConnFlags(cmd *cobra.Command, f *connFlags) {
	cmd.Flags().StringVar(&f.apiKey, "api-key", "", "Override API key.")
	_ = cmd.MarkFlagFilename("api-key")
	cmd.Flags().StringVar(&f.baseURL, "base-url", "", "Override API base URL.")
}

type scopeFlags struct {
	userID  string
	agentID string
	appID   string
	runID   string
}

func addScopeFlags(cmd *cobra.Command, f *scopeFlags) {
	cmd.Flags().StringVarP(&f.userID, "user-id", "u", "", "Scope to user.")
	cmd.Flags().StringVar(&f.agentID, "agent-id", "", "Scope to agent.")
	cmd.Flags().StringVar(&f.appID, "app-id", "", "Scope to app.")
	cmd.Flags().StringVar(&f.runID, "run-id", "", "Scope to run.")
}

func (f *scopeFlags) ids() EntityIDs {
	return EntityIDs{UserID: f.userID, AgentID: f.agentID, AppID: f.appID, RunID: f.runID}
}

// getBackendAndConfig 对齐 _get_backend_and_config: flag > env > file; ping 校验。
func getBackendAndConfig(apiKey, baseURL string) (*Config, Backend, error) {
	cfg := LoadConfig()
	if apiKey != "" {
		cfg.Platform.APIKey = apiKey
	}
	if baseURL != "" {
		cfg.Platform.BaseURL = baseURL
	}
	if cfg.Platform.APIKey == "" {
		printError("No API key configured.", "Run 'memgo init' or set MEM0_API_KEY environment variable.")
		return nil, nil, errExit
	}
	backend := GetBackend(cfg)
	if _, err := backend.Ping(timeout5s()); err != nil {
		if ae, ok := err.(*APIError); ok && ae.Kind == "auth" {
			printError("Invalid or expired API key.", "Run 'memgo init' or set MEM0_API_KEY environment variable.")
			return nil, nil, errExit
		}
		printInfo("Could not validate API key (network issue). Proceeding anyway.")
	} else if pingData, err := backend.Ping(timeout5s()); err == nil {
		if email := strOf(pingData["user_email"]); email != "" && cfg.Platform.UserEmail != email {
			cfg.Platform.UserEmail = email
			_ = SaveConfig(cfg)
		}
	}
	return cfg, backend, nil
}
