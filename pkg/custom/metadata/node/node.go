package NodeData

import (
    "sort"
	"github.com/apache/yunikorn-core/pkg/scheduler/objects"
)

type NodeData struct {
	nodeIDs        	[]string
	nodeRefs 		[]*objects.Node
	nodeCount      	int

	resourceCount  	int
	resourceTypes  	[]string

	nodeLimits 		[][]float64
	totalLimit    	[]float64
}

func NewNodeData(resourceTypes []string) *NodeData {
	return &NodeData{
		nodeIDs: 		make([]string, 0),
		nodeRefs:		make([]*objects.Node, 0),
		nodeCount: 		0,

		resourceCount:	len(resourceTypes),
		resourceTypes:	resourceTypes,

		nodeLimits:		make([][]float64, 0),
		totalLimit:		make([]float64, len(resourceTypes)),
	}
}

func (nodeData *NodeData) GetNodeId(index int) string{
	return nodeData.nodeIDs[index]
}
func (nodeData *NodeData) UpdateLimits() {
	nodeLimits := make([][]float64, 0)
	totalLimit := make([]float64, nodeData.resourceCount)

	for _, node := range nodeData.nodeRefs {
		limits := make([]float64, 0)
		nodeAvaiResources := *node.GetAvailableResource()

		for i, resourceType := range nodeData.resourceTypes {
			resourceValue := float64(nodeAvaiResources.Resources[resourceType])

			limits = append(limits, resourceValue)

			totalLimit[i] += resourceValue
		}
		nodeLimits = append(nodeLimits, limits)
	}

	nodeData.nodeLimits = nodeLimits
	nodeData.totalLimit = totalLimit
}

func (nodeData *NodeData) GetNodeCount() int {
	return nodeData.nodeCount
}

func (nodeData *NodeData) GetNodeLimits() [][]float64{
	return nodeData.nodeLimits
}

func (nodeData *NodeData) GetTotalLimit() []float64{
	return nodeData.totalLimit
}

// Parse the vcore and memory in node
func (nodeData *NodeData) AddNode(n *objects.Node) {
	if n.NodeID == "yk0" {
		return 
	}

	nodeData.nodeIDs 	= append(nodeData.nodeIDs, n.NodeID)
	sort.Strings(nodeData.nodeIDs)
	nodeData.nodeRefs	= append(nodeData.nodeRefs, n)
	nodeData.nodeCount	= len(nodeData.nodeRefs);
	
	nodeData.UpdateLimits()
}