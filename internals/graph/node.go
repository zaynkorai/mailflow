package graph

import (
	"context"
	"mailflow/internals/common/types"
)

// Node represents a step or an agent in the workflow graph.
// Each node has a unique ID and can execute a specific operation.
type Node interface {
	ID() string // Returns the unique identifier of the node
	// Execute performs the operation associated with this node.
	// It takes the current task and workflow context, and returns the updated task.
	Execute(ctx context.Context, task types.Task, wfCtx *types.WorkflowContext) (types.Task, error) // Use types.Task and types.WorkflowContext
}

// ConditionalRouterNode is a special type of node that can determine the next node based on the task state.
type ConditionalRouterNode interface {
	Node
	// Route determines the next node ID based on the task's current state/result.
	// It returns the ID of the next node and any error.
	Route(ctx context.Context, task types.Task, wfCtx *types.WorkflowContext) (string, error) // Use types.Task and types.WorkflowContext
}
