package NodeData

import (
	"github.com/apache/yunikorn-core/pkg/scheduler/objects"
)

type NodeData struct {
	NodeCount      int
	NodeIDs        []string
	Nodes 		   []*objects.Node
	ResourceCount  int
	ResourceTypes  []string
	ResourceLimits [][]float64
	TotalLimits    []float64
}

func NewNodeData(ResourceTypes []string) *NodeData {
	ResourceCount := len(ResourceTypes)
	return &NodeData{
		NodeCount: 0,
		NodeIDs: make([]string, 0),
		ResourceCount: len(ResourceTypes),
		ResourceTypes:   ResourceTypes,
		ResourceLimits: make([][]float64, 0),
		TotalLimits:    make([]float64, ResourceCount),
	}
}

func (nodeData *NodeData) UpdateLimits() {
	nodeLimits := make([][]float64, 0)
	totalLimit := make([]float64, 2)

	for _, node := range nodeData.Nodes {
		limits := make([]float64, 0)
		nodeAvaiResources := *node.GetAvailableResource()

		for i, resourceType := range nodeData.ResourceTypes {
			resourceValue := float64(nodeAvaiResources.Resources[resourceType])

			limits = append(limits, resourceValue)

			totalLimit[i] += resourceValue
		}
		nodeLimits = append(nodeLimits, limits)
	}

	nodeData.ResourceLimits = nodeLimits
	nodeData.TotalLimits = totalLimit
}

func (nodeData *NodeData) GetNodeLimits() [][]float64{
	return nodeData.ResourceLimits
}

func (nodeData *NodeData) GetTotalLimits() []float64{
	return nodeData.TotalLimits
}

// Parse the vcore and memory in node
func (nodeData *NodeData) AddNode(n *objects.Node) {
	if n.NodeID == "yk0" {
		return 
	}
	nodeData.Nodes = append(nodeData.Nodes, n)
	nodeData.NodeIDs = append(nodeData.NodeIDs, n.NodeID)
	nodeData.NodeCount += 1;
	
	nodeData.UpdateLimits()
}

// make test easy 
func (nodeData *NodeData) AddNodeDirectly(nodeName string, resource []float64) {
	nodeData.NodeCount += 1
	
	availableLimit := make([]float64, 2)
	for index := 0; index < len(resource); index++ {
		availableLimit[index] += resource[index]
		nodeData.TotalLimits[index] += availableLimit[index]
	}

	nodeData.ResourceLimits = append(nodeData.ResourceLimits, availableLimit)
}