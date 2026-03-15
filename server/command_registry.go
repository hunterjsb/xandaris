package server

import (
	"fmt"

	"github.com/hunterjsb/xandaris/game"
)

// CommandHandler processes a single game command.
type CommandHandler func(cmd game.GameCommand)

// CommandRegistry dispatches game commands to registered handlers.
type CommandRegistry struct {
	handlers map[game.CommandType]CommandHandler
}

// NewCommandRegistry creates an empty registry.
func NewCommandRegistry() *CommandRegistry {
	return &CommandRegistry{
		handlers: make(map[game.CommandType]CommandHandler),
	}
}

// Register adds a handler for a command type.
func (cr *CommandRegistry) Register(t game.CommandType, h CommandHandler) {
	cr.handlers[t] = h
}

// Execute dispatches a command to its registered handler.
// Returns an error if no handler is registered for the command type.
func (cr *CommandRegistry) Execute(cmd game.GameCommand) error {
	h, ok := cr.handlers[cmd.Type]
	if !ok {
		return fmt.Errorf("unknown command type: %s", cmd.Type)
	}
	h(cmd)
	return nil
}
