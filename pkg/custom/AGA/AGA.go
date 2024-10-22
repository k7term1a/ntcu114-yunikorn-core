package AGA

import (
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	Metadata "github.com/apache/yunikorn-core/pkg/custom/metadata"
	"github.com/apache/yunikorn-core/pkg/metrics"
	agamath "github.com/apache/yunikorn-core/pkg/custom/math"

	"github.com/apache/yunikorn-core/pkg/custom/ACO"
	"github.com/apache/yunikorn-core/pkg/custom/GOA"

	"github.com/apache/yunikorn-core/pkg/custom/math/vector"

	"github.com/apache/yunikorn-core/pkg/log"
	"github.com/apache/yunikorn-core/pkg/scheduler/objects"
)

var (
	aga *AGA
	startTime time.Time
	lastDuration = 0.0

	allZeroResult = 0
	decisionResult = 0
) 

type AGA struct {
	metadata 	*Metadata.Metadata
	goa			*GOA.GOA
	aco 		*ACO.ACO

	sync.RWMutex
}

func Init() {
	aga = NewAGA()
	lastDuration = 0.0

	allZeroResult = 0
	decisionResult = 0
}

func GetAGA() *AGA {
	return aga
}

func NewAGA() *AGA{
	return &AGA{
		metadata: Metadata.NewMetadata(),
		goa: GOA.NewGOA(),
		aco: ACO.NewACO(),
	}
}

func (aga *AGA) randanInitValue(amount int) []*vector.Vector{
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	users := aga.metadata.UserData.UserCount
	nodes := aga.metadata.NodeData.NodeCount

	candidates := make([]*vector.Vector, 0)

	d := make([]int, users)
	for j := 0; j < users; j++ {
		userName := aga.metadata.UserData.GetName(j)
		count := aga.metadata.UserData.GetUserAskCount(userName)
		d[j] = count;
	}
	log.Log(log.Custom).Info(fmt.Sprintf("remain: %v", d))

	for i := 0; i < amount; i++ {
		candidateArray := make([]int, users*nodes)
		distributeAmount := make([]int, users)

		for j := 0; j < users; j++ {
			userName := aga.metadata.UserData.GetName(j)
			count := aga.metadata.UserData.GetUserAskCount(userName)
			distributeAmount[j] = count;
		}
		// log.Log(log.Custom).Info(fmt.Sprintf("%v", distributeAmount))

		for j := 0; j < len(candidateArray); j++ {
			userIndex := j % users;
			remains := distributeAmount[userIndex]
			tmp := int64(float64(remains) * float64(i) / float64(amount))
			// log.Log(log.Custom).Info(fmt.Sprintf("%v * %v = %v", remains, float64(i) / float64(aga.aco.HyperParameter.AntNum), tmp))
			if tmp <= 0 {
				tmp = 1
			}
			// log.Log(log.Custom).Info(fmt.Sprintf("%v", tmp))
			amount := r.Int63n(tmp)

			candidateArray[j] = int(amount)
			distributeAmount[userIndex] -= int(amount)
		}

		candidate := vector.NewVectorByInt(candidateArray)
		candidates = append(candidates, candidate)

		// log.Log(log.Custom).Info(fmt.Sprintf("candidate score is %v", agamath.GetScore(aga.metadata, candidate)))

	} 
	return candidates
}

func (aga *AGA) Start() []int{

	users := aga.metadata.UserData.UserCount;
	nodes := aga.metadata.NodeData.NodeCount;

	ACOParameter := ACO.NewACOHyperParameter(ACO_NumAnt, ACO_Epochs, users*nodes * ACO_Steps)
	GOAParameter := GOA.NewGOAHyperParameter(
			GOA_iterations, 
			GOA_cMax, 
			GOA_cMin, 
			GOA_grasshopperAmount,	
			GOA_GForce, 
			GOA_WindForce, 
	)
	aga.aco.SetHyperParameter(ACOParameter)
	aga.aco.SetMetadata(aga.metadata)

	aga.goa.SetHyperParameter(GOAParameter)
	aga.goa.SetMetaData(aga.metadata)

	aga.metadata.CalculateDRs()

	aga.aco.Start(aga.randanInitValue(ACO_NumAnt))

	desision := aga.goa.Start(aga.aco.GetCandidate(GOA_grasshopperAmount))

	return desision
}

func (aga *AGA) ACOStart() []int{

	users := aga.metadata.UserData.UserCount;
	nodes := aga.metadata.NodeData.NodeCount;

	ACOParameter := ACO.NewACOHyperParameter(50, 50, users*nodes * ACO_Steps)

	aga.aco.SetHyperParameter(ACOParameter)
	aga.aco.SetMetadata(aga.metadata)
	aga.metadata.CalculateDRs()

	aga.aco.Start(aga.randanInitValue(50))
	decision := aga.aco.GetCandidate(1)[0].ToIntArray()

	return decision
}

func (aga *AGA) GOAStart() []int{
	GOAParameter := GOA.NewGOAHyperParameter(
			20, 
			GOA_cMax, 
			GOA_cMin, 
			50,	
			GOA_GForce, 
			GOA_WindForce, 
	)
	aga.goa.SetHyperParameter(GOAParameter)
	aga.goa.SetMetaData(aga.metadata)
	aga.metadata.CalculateDRs()

	desision := aga.goa.Start(aga.randanInitValue(50))

	return desision
}

func (aga *AGA) metricUpdateNodeUsage() {
	for id, limits := range aga.metadata.GetNodeLimits() {
		metrics.GetCustomMetrics().SetCustomCPUUsage(fmt.Sprintf("node_%v", id), limits[0])
		metrics.GetCustomMetrics().SetCustomMemoryUsage(fmt.Sprintf("node_%v", id), limits[1])
	}

}

func (aga *AGA) GetAllocations() (allocs []*objects.Allocation) {
	
	aga.Lock()
	defer aga.Unlock()
	lastDuration = 0.0
	startTime = time.Now()

	allocs = make([]*objects.Allocation, 0)

	users := aga.metadata.UserData.UserCount
	nodes := aga.metadata.NodeData.NodeCount

	aga.metadata.UpdateLimits()
	aga.metricUpdateNodeUsage()

	if users * nodes == 0 {
		return 
	}

	log.Log(log.Custom).Info("AGA start")

	decision := aga.Start()

	
	decisionResult += 1

	result := make([]int, users)

	for nodeIndex, nodeId := range aga.metadata.Nodes {
		for userIndex := 0; userIndex < users; userIndex++ {
			if distributeAmount := decision[nodeIndex*users+userIndex]; distributeAmount != 0 {
				
				name := aga.metadata.UserData.GetName(userIndex)
				asks := aga.metadata.UserData.PopAsks(name, distributeAmount)
				for _, ask := range asks {
					alloc := objects.NewAllocation(nodeId, ask)
					allocs = append(allocs, alloc)
				}
				result[userIndex] += len(asks)
			}
		}
	}
	
	allZero := 1
	s := ""
	for i, num := range result {
		if num != 0 {
			allZero = 0
		}
		if i == 0 {
			s += fmt.Sprintf("user result is: %v", num) 
		} else {
			s += fmt.Sprintf(", %v", num) 
		}
	}

	decisionResult += 1
	allZeroResult += allZero

	metrics.GetCustomMetrics().SetFinalZeroSolutionRatio(100.0 * (float64(allZeroResult) / float64(decisionResult)))

	finalScore := agamath.GetScore(aga.metadata, vector.NewVectorByInt(decision))
	if finalScore == math.Inf(1) {
		finalScore = -1
	}
	metrics.GetCustomMetrics().SetFinalDecisionScore(finalScore)

	log.Log(log.Custom).Info(s)
	lastDuration = time.Since(startTime).Seconds()

	aga.metadata.UserData.Update()
	return allocs
}

func GetLastDuration() float64{
	return lastDuration
}