package metadata

import (
	"math"
	"sync"

	"github.com/apache/yunikorn-core/pkg/scheduler/objects"

	NodeData "github.com/apache/yunikorn-core/pkg/custom/metadata/node"
	UserData "github.com/apache/yunikorn-core/pkg/custom/metadata/user"

	sicommon "github.com/apache/yunikorn-scheduler-interface/lib/go/common"
)

var (
	ResourceTypes = []string{sicommon.CPU, sicommon.Memory}
)

type Metadata struct {
	UserData           	*UserData.UserData
	NodeData           	*NodeData.NodeData
	DRs                	[][]float64
	DRRatioReciprocals 	[][]float64
	GlobalDRs                	[]float64
	GlobalDRRatioReciprocals 	[]float64

	*sync.RWMutex
}

func NewMetadata() *Metadata {
	return &Metadata{
		UserData:           UserData.NewUserData(ResourceTypes),
		NodeData:           NodeData.NewNodeData(ResourceTypes),
		DRs:                make([][]float64, 0),
		DRRatioReciprocals: make([][]float64, 0),
	}
}

func (metadata *Metadata) GetResourceCount() int{
	return len(ResourceTypes)
}

//users
func (metadata *Metadata) GetUserAskCount(index int) int{
	return metadata.UserData.GetUserAskCount(index)
}

func (metadata *Metadata) GetLastRequesti(index int) *objects.AllocationAsk {
	return metadata.UserData.GetLastRequest(index)
}

func (metadata *Metadata) GetUserCount() int {
	return metadata.UserData.GetUserCount()
}

func (metadata *Metadata) GetUserAsks() [][]float64 {
	return metadata.UserData.GetUserAsks()
}

func (metadata *Metadata) UpdateUserInfo(app []*objects.Application) {
	metadata.UserData.UpdateUserInfo(app)
}

// node
func (metadata *Metadata) GetNodeCount() int {
	return metadata.NodeData.GetNodeCount()
}

func (metadata *Metadata) GetNodeId(index int) string{
	return metadata.NodeData.GetNodeId(index)
}

func (metadata *Metadata) UpdateLimits() {
	metadata.NodeData.UpdateLimits()
}
func (metadata *Metadata) GetNodeLimits() [][]float64 {
	return metadata.NodeData.GetNodeLimits()
}

func (metadata *Metadata) GetTotalLimit() []float64 {
	return metadata.NodeData.GetTotalLimit()
}

func (metadata *Metadata) AddNode(node *objects.Node) {
	metadata.NodeData.AddNode(node)
}

func (metadata *Metadata) CalculateDRs() {
	metadata.DRs = make([][]float64, 0)
	metadata.DRRatioReciprocals = make([][]float64, 0)

	for _, limit := range metadata.GetNodeLimits() {
		DR := make([]float64, 0)
		DRRatioReciprocal := make([]float64, 0)

		userAsks := metadata.UserData.GetUserAsks()
		for _, askResources := range userAsks {
			maxRatio := 0.0

			for i := 0; i < len(ResourceTypes); i++ {
				ratio := askResources[i] / limit[i]
				if ratio > maxRatio {
					maxRatio = ratio
				}
			}

			DR = append(DR, maxRatio)
			DRRatioReciprocal = append(DRRatioReciprocal, math.Pow(maxRatio, -1))
		}
		metadata.DRs = append(metadata.DRs, DR)
		metadata.DRRatioReciprocals = append(metadata.DRRatioReciprocals, DRRatioReciprocal)
	}

}

func (metadata *Metadata) CalculateGlobalDRs() {
	metadata.GlobalDRs = make([]float64, metadata.UserData.GetUserCount())
	metadata.GlobalDRRatioReciprocals = make([]float64, metadata.UserData.GetUserCount())

	totalLimit := metadata.GetTotalLimit()
	userAsks := metadata.UserData.GetUserAsks()
	for userIndex, askResources := range userAsks {
		maxRatio := 0.0
		for resourceIndex, limit := range totalLimit {
			ratio := askResources[resourceIndex] / limit
			if ratio > maxRatio {
				maxRatio = ratio
			}
		}
		metadata.GlobalDRs[userIndex] = maxRatio
		metadata.GlobalDRRatioReciprocals[userIndex] = math.Pow(maxRatio, -1)
	}
}