package schema

import (
	"context"
	"fmt"
)

// Chat schema
const (
	AIMsg MsgRole = iota
	UserMsg
	SysMsg
	ErrMsg
	InternalMsg
)

type MsgRole int

func (r MsgRole) String() string {
	switch r {
	case AIMsg:
		return "AIMsg"
	case UserMsg:
		return "UserMsg"
	case SysMsg:
		return "SysMsg"
	case ErrMsg:
		return "ErrMsg"
	case InternalMsg:
		return "InternalMsg"
	default:
		return fmt.Sprintf("MsgRole(%d)", r)
	}
}

func EnumRole(s string) MsgRole {
	switch s {
	case "AIMsg":
		return AIMsg
	case "UserMsg":
		return UserMsg
	case "SysMsg":
		return SysMsg
	case "ErrMsg":
		return ErrMsg
	case "InternalMsg":
		return InternalMsg
	default:
		return 0
	}
}

type Msg struct {
	Stream    bool
	Role      MsgRole
	Content   string
	Timestamp int64
}

type SessionStorage interface {
	NewSession() (ChatSession, error)
	ListSessions() []ChatSession
	LoadRecentSession() (ChatSession, error)
	LoadSession(string) (ChatSession, error)
	SaveSession(ChatSession) (ChatSession, error)
	DeleteSession(string) error
}

type ChatSession struct {
	ID        string
	Title     string
	Msgs      []Msg
	CreatedAt int64
}

type ChatProvider interface {
	Name() string
	Type() ProviderType
	Description() string
	Run(ctx context.Context, input string, session ChatSession) (string, error)
	Stream(ctx context.Context, callback func(ctx context.Context, msg Msg) error)
}

type ProviderType int

const (
	LLM ProviderType = iota
	Agent
	Workflow
)

func (t ProviderType) String() string {
	switch t {
	case LLM:
		return "LLM"
	case Agent:
		return "Agent"
	case Workflow:
		return "Workflow"
	default:
		return fmt.Sprintf("ProviderType(%d)", t)
	}
}
