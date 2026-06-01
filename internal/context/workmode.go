package context

import stdctx "context"

// Workmode injects the agent's work mode (default/yolo) into the system prompt.
type Workmode struct {
	Mode string
}

func (s *Workmode) Name() string { return "workmode" }
func (s *Workmode) Index() int   { return 2 }

func (s *Workmode) BuildContent(_ stdctx.Context) (string, error) {
	switch s.Mode {
	case "yolo":
		return `## Work Mode
You are in **yolo** mode. You can execute tools and actions autonomously without asking the user for permission. However, be careful — think before running commands, avoid destructive operations, and consider the consequences of your actions.`, nil
	default:
		return `## Work Mode
You are in **default** mode. Unless the user has explicitly pre-approved an action in this conversation, you MUST ask the user for permission before executing any tool or making changes. Wait for explicit confirmation before proceeding.`, nil
	}
}
