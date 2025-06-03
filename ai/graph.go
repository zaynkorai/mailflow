package ai

import (
	"context"
	"fmt"
	"log"

	"github.com/fatih/color"
)

// GraphEnd is a sentinel value indicating the end of a graph path.
const GraphEnd = "__END__"

// GraphNodeFunc defines the signature for a function that can be a node in our graph.
// It takes a context and the current state, returns the (potentially modified) state,
// a string for routing (empty if no routing, GraphEnd if path ends), and an error.
type GraphNodeFunc func(ctx context.Context, state *GraphState) (*GraphState, string, error)

// EdgeConfig defines how to transition from a source node.
type EdgeConfig struct {
	IsConditional  bool
	ToNode         string            // For direct edges
	RouterFunc     GraphNodeFunc     // The node function that determines the next step (for conditional edges)
	ConditionalMap map[string]string // Router output to next node name (for conditional edges)
}

type Graph struct {
	nodes      map[string]GraphNodeFunc
	edges      map[string]EdgeConfig
	entryPoint string
	nodesImpl  *Nodes
}

func NewGraph(nodesImpl *Nodes) *Graph {
	return &Graph{
		nodes:     make(map[string]GraphNodeFunc),
		edges:     make(map[string]EdgeConfig),
		nodesImpl: nodesImpl,
	}
}

// AddNode registers a node function with a given name.
func (g *Graph) AddNode(name string, nodeFunc GraphNodeFunc) {
	g.nodes[name] = nodeFunc
}

// SetEntryPoint sets the starting node of the graph.
func (g *Graph) SetEntryPoint(name string) {
	g.entryPoint = name
}

// AddEdge defines a direct (unconditional) transition from one node to another.
func (g *Graph) AddEdge(fromNode, toNode string) {
	g.edges[fromNode] = EdgeConfig{
		IsConditional: false,
		ToNode:        toNode,
	}
}

// AddConditionalEdges defines transitions based on the output of a router function.
func (g *Graph) AddConditionalEdges(fromNode string, routerFunc GraphNodeFunc, conditionalMap map[string]string) {
	g.edges[fromNode] = EdgeConfig{
		IsConditional:  true,
		RouterFunc:     routerFunc,
		ConditionalMap: conditionalMap,
	}
}

func (g *Graph) Compile() *Graph {
	return g
}

func (g *Graph) Execute(ctx context.Context, initialState GraphState, maxIterations int) (*GraphState, error) {
	currentState := initialState
	currentNodeName := g.entryPoint

	if _, ok := g.nodes[currentNodeName]; !ok {
		return nil, fmt.Errorf("entry point node '%s' not found", currentNodeName)
	}

	fmt.Printf("\n--- Starting Workflow Execution ---\nInitial State: %+v\n\n", currentState)

	for i := 0; i < maxIterations; i++ {
		if currentNodeName == GraphEnd {
			fmt.Println("Workflow reached END. Terminating.")
			break
		}

		fmt.Printf("Executing node: %s\n", currentNodeName)

		nodeFunc, ok := g.nodes[currentNodeName]
		if !ok {
			return nil, fmt.Errorf("node '%s' not found in graph definition", currentNodeName)
		}

		// First, execute the node itself.
		// For a non-routing node, `nodeDecision` will likely be empty.
		// For a node that *is* a router (like CheckNewEmails or RouteEmailBasedOnCategory)
		// it will be the routing decision.
		updatedState, _, err := nodeFunc(ctx, &currentState)
		if err != nil {
			return nil, fmt.Errorf("error executing node '%s': %w", currentNodeName, err)
		}
		currentState = *updatedState

		fmt.Println(color.CyanString("Finished running: %s", currentNodeName))

		edgeConfig, edgeExists := g.edges[currentNodeName]
		if !edgeExists {
			fmt.Printf("Node '%s' has no outgoing edges. Implicitly ending path.\n", currentNodeName)
			currentNodeName = GraphEnd
			continue
		}

		var routingDecision string
		if edgeConfig.IsConditional {
			// If the edge from this node is conditional, then we call the specific RouterFunc
			_, decisionFromRouterFunc, routerErr := edgeConfig.RouterFunc(ctx, &currentState)
			if routerErr != nil {
				return nil, fmt.Errorf("error executing router function for node '%s': %w", currentNodeName, routerErr)
			}
			routingDecision = decisionFromRouterFunc

			fmt.Printf("Node '%s' is conditional. Router function decided: '%s'\n", currentNodeName, routingDecision)

			nextNode, ok := edgeConfig.ConditionalMap[routingDecision]
			if !ok {
				return nil, fmt.Errorf("conditional edge from '%s' has no mapping for decision '%s'", currentNodeName, routingDecision)
			}
			currentNodeName = nextNode
		} else {
			// For a direct edge, there's no router function, just a direct `ToNode`.
			currentNodeName = edgeConfig.ToNode
		}

		fmt.Printf("Transitioning to node: %s\n\n", currentNodeName)

		if i == maxIterations-1 && currentNodeName != GraphEnd {
			log.Printf("Warning: Workflow reached max iterations (%d) without reaching END. Terminating to prevent infinite loop.\n", maxIterations)
			break
		}
	}

	fmt.Printf("\n--- Workflow Execution Finished ---\nFinal State: %+v\n", currentState)
	return &currentState, nil
}
