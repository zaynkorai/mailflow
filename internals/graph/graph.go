package graph

import (
	"context"
	"fmt"
	"mailflow/internals/common/types"
	"mailflow/pkg/logging"
	"sync"
)

// Directed graph.
type Graph struct {
	nodes            map[string]Node
	edges            map[string][]string          // Adjacency list: nodeID -> list of nextNodeIDs
	conditionalEdges map[string]map[string]string // nodeID -> (routeKey -> nextNodeID)
	startNodeID      string
	mu               sync.RWMutex
	// CompiledGraph stores the compiled, immutable version of the graph for execution.
	// This is a placeholder for a more complex compilation step if needed.
	CompiledGraph *Graph
}

const GraphEnd = "GRAPH_END"

func NewGraph() *Graph {
	return &Graph{
		nodes:            make(map[string]Node),
		edges:            make(map[string][]string),
		conditionalEdges: make(map[string]map[string]string),
	}
}

func (g *Graph) AddNode(node Node) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if _, exists := g.nodes[node.ID()]; exists {
		return fmt.Errorf("node with ID '%s' already exists", node.ID())
	}
	g.nodes[node.ID()] = node
	g.edges[node.ID()] = []string{}
	logging.Debug("Added node: %s", node.ID())
	return nil
}

func (g *Graph) AddEdge(fromNodeID, toNodeID string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if _, exists := g.nodes[fromNodeID]; !exists {
		return fmt.Errorf("source node '%s' not found", fromNodeID)
	}

	if toNodeID != GraphEnd {
		if _, exists := g.nodes[toNodeID]; !exists {
			return fmt.Errorf("destination node '%s' not found", toNodeID)
		}
	}

	// Prevent duplicate edges
	for _, existingTo := range g.edges[fromNodeID] {
		if existingTo == toNodeID {
			return fmt.Errorf("edge from '%s' to '%s' already exists", fromNodeID, toNodeID)
		}
	}

	g.edges[fromNodeID] = append(g.edges[fromNodeID], toNodeID)
	logging.Debug("Added edge: %s -> %s", fromNodeID, toNodeID)
	return nil
}

func (g *Graph) AddConditionalEdges(routerNodeID string, router ConditionalRouterNode, routes map[string]string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if _, exists := g.nodes[routerNodeID]; !exists {
		return fmt.Errorf("router node '%s' not found", routerNodeID)
	}

	// Ensure the routerNodeID is associated with a ConditionalRouterNode
	if _, ok := g.nodes[routerNodeID].(ConditionalRouterNode); !ok {
		return fmt.Errorf("node '%s' is not a ConditionalRouterNode", routerNodeID)
	}

	g.conditionalEdges[routerNodeID] = routes
	logging.Debug("Added conditional edges for router node '%s': %v", routerNodeID, routes)

	// Validate all destination nodes exist (except for GraphEnd)
	for _, destNodeID := range routes {
		if destNodeID != GraphEnd {
			if _, exists := g.nodes[destNodeID]; !exists {
				return fmt.Errorf("conditional destination node '%s' not found", destNodeID)
			}
		}
	}
	return nil
}

func (g *Graph) SetStartNode(nodeID string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if _, exists := g.nodes[nodeID]; !exists {
		return fmt.Errorf("start node '%s' not found in graph", nodeID)
	}
	g.startNodeID = nodeID
	logging.Info("Set start node to: %s", nodeID)
	return nil
}

// Compile "compiles" the graph. For this simple implementation, it just returns itself.
// In a more complex system, this might involve validation, optimization, or creating an immutable execution plan.
func (g *Graph) Compile() *Graph {
	logging.Info("Compiling graph...")
	// [TODO]Perform validation here if needed (e.g., check for cycles, unreachable nodes)
	return g // For now, just return the graph itself
}

func (g *Graph) Execute(ctx context.Context, initialTask types.Task, wfCtx *types.WorkflowContext) (types.Task, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if g.startNodeID == "" {
		return initialTask, fmt.Errorf("no start node defined for the graph")
	}

	currentNodeID := g.startNodeID
	currentTask := initialTask

	logging.Info("Starting graph execution for task ID: %s, from node: %s", currentTask.ID, currentNodeID)

	for {
		select {
		case <-ctx.Done():
			logging.Info("Graph execution cancelled for task %s: %v", currentTask.ID, ctx.Err())
			return currentTask, ctx.Err()
		default:
			if currentNodeID == GraphEnd {
				logging.Info("Reached GRAPH_END sentinel for task %s.", currentTask.ID)
				return currentTask, nil // Workflow completed successfully
			}

			node, exists := g.nodes[currentNodeID]
			if !exists {
				return currentTask, fmt.Errorf("node '%s' not found during execution for task %s", currentNodeID, currentTask.ID)
			}

			logging.Info("Executing node '%s' for task %s (Current State: %s)", node.ID(), currentTask.ID, currentTask.CurrentState)
			updatedTask, err := node.Execute(ctx, currentTask, wfCtx)
			if err != nil {
				logging.Error("Error executing node '%s' for task %s: %v", node.ID(), currentTask.ID, err)
				updatedTask.Error = err // Store the error in the task
				return updatedTask, fmt.Errorf("node '%s' failed: %w", node.ID(), err)
			}
			currentTask = updatedTask
			logging.Info("Node '%s' completed for task %s (New State: %s)", node.ID(), currentTask.ID, currentTask.CurrentState)

			// Determine next node
			if routes, isConditional := g.conditionalEdges[currentNodeID]; isConditional {
				// This is a conditional router node
				routerNode, ok := node.(ConditionalRouterNode)
				if !ok {
					return currentTask, fmt.Errorf("node '%s' is defined as conditional but does not implement ConditionalRouterNode", currentNodeID)
				}
				routeKey, err := routerNode.Route(ctx, currentTask, wfCtx)
				if err != nil {
					return currentTask, fmt.Errorf("failed to route from node '%s': %w", currentNodeID, err)
				}
				next, found := routes[routeKey]
				if !found {
					return currentTask, fmt.Errorf("no route found for key '%s' from node '%s'", routeKey, currentNodeID)
				}
				currentNodeID = next
				logging.Debug("Conditional routing from '%s' to '%s' via key '%s'", node.ID(), currentNodeID, routeKey)
			} else {
				// This is a sequential node
				nextNodes := g.edges[currentNodeID]
				if len(nextNodes) == 0 {
					logging.Info("Reached end of sequential path for task %s at node %s.", currentTask.ID, currentNodeID)
					return currentTask, nil // Workflow completed
				}
				currentNodeID = nextNodes[0] // Move to the next node (first edge)
				logging.Debug("Sequential routing from '%s' to '%s'", node.ID(), currentNodeID)
			}
		}
	}
}
