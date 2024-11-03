package custom

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	customMath "github.com/apache/yunikorn-core/pkg/custom/math"
	"github.com/apache/yunikorn-core/pkg/custom/math/vector"
	Metadata "github.com/apache/yunikorn-core/pkg/custom/metadata"

	"github.com/apache/yunikorn-core/pkg/custom/ACO"
	"github.com/apache/yunikorn-core/pkg/custom/AGA"
	"github.com/apache/yunikorn-core/pkg/custom/GOA"

	"github.com/apache/yunikorn-core/pkg/metrics"
	"github.com/apache/yunikorn-core/pkg/log"
	"github.com/apache/yunikorn-core/pkg/scheduler/objects"
)

var (
	aga					*AGA.AGA
	aco 				*ACO.ACO
	goa 				*GOA.GOA

	metadata			*Metadata.Metadata
	ACOParameter 		*ACO.ACOHyperParameter
	GOAParameter		*GOA.GOAHyperParameter

	// for metrics
	decisionResult 		float64
	allZeroResult 		float64
	lastDuration		float64
) 

func Init() {
	aga = AGA.NewAGA()
	aco = ACO.NewACO()
	goa = GOA.NewGOA()

	metadata = Metadata.NewMetadata()

	log.Log(log.Custom).Info("custom algorithm start")
}

func AddNode(n *objects.Node) {
	metadata.AddNode(n)
}

func GetPendingApps(apps []*objects.Application) (pendingApps []*objects.Application){
	pendingApps = make([]*objects.Application, 0)
	for _, app := range apps {
		requests := app.GetAllRequests()
		pendingCount := 0
		for _, request := range requests {
			pendingCount += int(request.GetPendingAskRepeat())
		}
		log.Log(log.Custom).Info(fmt.Sprintf("count is %v", pendingCount))
		if pendingCount != 0 {
			pendingApps = append(pendingApps, app)
		}
	}
	return 
}

func randanInitValue(amount int) []*vector.Vector{
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	users := metadata.GetUserCount()
	nodes := metadata.GetNodeCount()

	candidates := make([]*vector.Vector, 0)

	// get max allocation amount
	d := make([]int, users)
	for userIndex := 0; userIndex < users; userIndex++ {
		count := metadata.GetUserAskCount(userIndex)
		d[userIndex] = count;
	}
	log.Log(log.Custom).Info(fmt.Sprintf("remain: %v", d))

	// random from remain amount
	for i := 0; i < amount; i++ {
		candidateArray := make([]int, users*nodes)
		distributeAmount := make([]int, users)

		for userIndex := 0; userIndex < users; userIndex++ {
			count := metadata.GetUserAskCount(userIndex)
			distributeAmount[userIndex] = count;
		}

		for j := 0; j < len(candidateArray); j++ {
			userIndex := j % users;
			remains := distributeAmount[userIndex]
			tmp := int64(float64(remains) * float64(i) / float64(amount))
			if tmp <= 0 {
				tmp = 1
			}
			amount := r.Int63n(tmp)

			candidateArray[j] = int(amount)
			distributeAmount[userIndex] -= int(amount)
		}

		candidate := vector.NewVectorByInt(candidateArray)
		candidates = append(candidates, candidate)
	} 
	return candidates
}

func metricUpdateUsage() {
	for id, limits := range metadata.GetNodeLimits() {
		metrics.GetCustomMetrics().SetCustomCPUUsage(fmt.Sprintf("node_%v", id), limits[0])
		metrics.GetCustomMetrics().SetCustomMemoryUsage(fmt.Sprintf("node_%v", id), limits[1])
	}
}

func InitHyperParameter() {
	users := metadata.GetUserCount()
	nodes := metadata.GetNodeCount()

	ACOParameter = ACO.NewACOHyperParameter(ACO_NumAnt, ACO_Epochs, users*nodes * ACO_Steps)
	GOAParameter = GOA.NewGOAHyperParameter(
			GOA_iterations, 
			GOA_cMax, 
			GOA_cMin, 
			GOA_grasshopperAmount,	
			GOA_GForce, 
			GOA_WindForce, 
	)
}

func AGAStart() (decision []int){
	aga.SetMetaData(metadata)
	candidates := randanInitValue(ACOParameter.AntNum)
	decision = aga.Start(candidates, ACOParameter, GOAParameter)
	return 
}

func ACOStart() (decision []int){
	aco.SetMetaData(metadata)
	candidates := randanInitValue(ACOParameter.AntNum)
	decision = aga.Start(candidates, ACOParameter, GOAParameter)
	return 
}

func GOAStart() (decision []int){
	goa.SetMetaData(metadata)
	candidates := randanInitValue(GOAParameter.GrasshopperAmount)
	decision = aga.Start(candidates, ACOParameter, GOAParameter)
	return 
}


func GetAllocations(app []*objects.Application, alogorithmIndex int) (fakeAllocs[]*objects.Allocation){
	startTime := time.Now()

	metadata.UpdateLimits()
	metadata.UpdateUserInfo(app)
	metricUpdateUsage()

	users := metadata.GetUserCount()
	nodes := metadata.GetNodeCount()

	if users * nodes == 0 {
		return nil
	}

	InitHyperParameter()

	var decision []int
	if alogorithmIndex == 0 {
		decision = AGAStart()
	} else if alogorithmIndex == 1 {
		decision = ACOStart()
	} else {
		decision = GOAStart()
	}

	log.Log(log.Custom).Info(fmt.Sprintf("decision is %v", decision))

	finalScore := customMath.GetScore(metadata, vector.NewVectorByInt(decision))
	checkAllZero(decision)

	for nodeIndex := 0; nodeIndex < nodes; nodeIndex++ {
		nodeID := metadata.GetNodeId(nodeIndex)
		for userIndex := 0; userIndex < users; userIndex++ {
			if distributeAmount := decision[nodeIndex*users+userIndex]; distributeAmount != 0 {

				for disIndex := 0; disIndex < distributeAmount; disIndex++ {
					ask := metadata.GetLastRequesti(userIndex)
					fakeAlloc := objects.NewAllocation(nodeID, ask)
					fakeAllocs = append(fakeAllocs, fakeAlloc)
				}
			}
		}
	}


	if finalScore == math.Inf(1) {
		finalScore = -1
	}
	metrics.GetCustomMetrics().SetFinalDecisionScore(finalScore)
	lastDuration = time.Since(startTime).Seconds()
	return 
}

func GetLastDuration() float64{
	return lastDuration
}

func checkAllZero(decision []int) {
	users := metadata.GetUserCount()
	nodes := metadata.GetNodeCount()
	
	result := make([]int, users)

	for nodeIndex := 0; nodeIndex < nodes; nodeIndex++ {
		for userIndex := 0;userIndex < users; userIndex++ {
			if distributeAmount := decision[nodeIndex*users+userIndex]; distributeAmount != 0 {
				result[userIndex] += distributeAmount
			}
		}
	}

	allZero := 1.0
	s := ""
	for i, num := range result {
		if num != 0 {
			allZero = 0.0
		}
		if i == 0 {
			s += fmt.Sprintf("user result is: %v", num) 
		} else {
			s += fmt.Sprintf(", %v", num) 
		}
	}

	log.Log(log.Custom).Info(s)

	decisionResult += 1
	allZeroResult += allZero

	metrics.GetCustomMetrics().SetFinalZeroSolutionRatio(100.0 * (float64(allZeroResult) / float64(decisionResult)))
}