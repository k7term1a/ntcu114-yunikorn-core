package math

import (
	// "fmt"
	"math"

	"github.com/apache/yunikorn-core/pkg/custom/math/vector"
	Metadata "github.com/apache/yunikorn-core/pkg/custom/metadata"
	// "github.com/apache/yunikorn-core/pkg/log"
)

func GetEffectScore(metadata *Metadata.Metadata, candidate *vector.Vector) float64 {
	resources := metadata.GetResourceCount()
	nodes := metadata.GetNodeCount()
	users := metadata.GetUserCount()
	
	userTotal := make([]float64, resources)

	for i := 0; i < nodes; i++ {
		totalAtNodeI := make([]float64, resources)
		for j := 0; j < users; j++ {
			
			amountThatUserJTakeAtNodeI := int(candidate.Get(i * users + j))

			if amountThatUserJTakeAtNodeI < 0 {
				return math.Inf(1)
			} 
			for k := 0; k < resources; k++ {
				nodeIOccupied := float64(amountThatUserJTakeAtNodeI) * metadata.GetUserAsks()[j][k]
				totalAtNodeI[k] += nodeIOccupied

				if totalAtNodeI[k] > metadata.GetNodeLimits()[i][k] {
					return math.Inf(1)
				}
				userTotal[k] += nodeIOccupied
			}
		}
	}

	totalLimitVector := vector.NewVector(metadata.GetTotalLimit())
	userTotalVector := vector.NewVector(userTotal)
	totalLimitsDistance := totalLimitVector.Norm()
	userTotalDistance := userTotalVector.Norm()
	
	ratio := userTotalDistance / totalLimitsDistance

	// 計算「空閒資源比例」。使用「使用者資源上限」除以「機器資源上限」，(1 - 所得比例) * 100 為「空閒資源佔比」
	percentage := (1 - ratio) * 100

	return percentage
}

func GetBalanceScore(metadata *Metadata.Metadata, candidate *vector.Vector) float64 {
	nodes := metadata.GetNodeCount()
	users := metadata.GetUserCount()

	occupied := make([]float64, nodes)

	for i := 0; i < nodes; i++ {
		occupiedAtNodeI := make([]float64, metadata.GetResourceCount())
		for j := 0; j < users; j++{
			amountThatUserJTakeAtNodeI := candidate.Get(i*users+j)
			if amountThatUserJTakeAtNodeI < 0 {
				return math.Inf(1)
			}
			cpuOccupy := amountThatUserJTakeAtNodeI * metadata.UserData.GetUserAsk(j)[0]
			occupiedAtNodeI[0] += cpuOccupy
			memOccupy := amountThatUserJTakeAtNodeI * metadata.UserData.GetUserAsk(j)[1]
			occupiedAtNodeI[1] += memOccupy
		}

		limitOfNodeI := metadata.GetNodeLimits()[i]
		for resource := 0; resource < metadata.GetResourceCount(); resource++ {
			if occupiedAtNodeI[resource] > limitOfNodeI[resource]{
				return math.Inf(1)
			}else {
				occupied[i] = math.Max(occupied[i], float64(occupiedAtNodeI[resource] / limitOfNodeI[resource]))
			}
		}
	}
	jainIndexValue := math.Pow(Sum(occupied), 2) / SumOfSquares(occupied) * float64(nodes)

	return jainIndexValue
	
}

func GetFairnessScore(metadata *Metadata.Metadata, candidate *vector.Vector) float64 {
	nodes := metadata.GetNodeCount()
	users := metadata.GetUserCount()

	usersDR := make([]float64, users)

	for i := 0; i < nodes; i++ {
		for j := 0; j < users; j++ {
			amountThatUserJTakeAtNodeI := candidate.Get(i*users+j)
			if amountThatUserJTakeAtNodeI < 0 {
				return math.Inf(1)
			}
			usersDR[j] += float64(amountThatUserJTakeAtNodeI) * metadata.DRs[i][j]
		}
	}
	if Sum(usersDR) == 0 {
		return math.Inf(1)
	}

	jainIndexValue := math.Pow(Sum(usersDR), 2) / (SumOfSquares(usersDR) * float64(users))

	return jainIndexValue
}

func GetScore(metadata *Metadata.Metadata, candidate *vector.Vector) float64 {
	effectScore := GetEffectScore(metadata, candidate)
	if effectScore == math.Inf(1) {
		return math.Inf(1)
	}
	fairnessScore := GetFairnessScore(metadata, candidate)
	if fairnessScore == math.Inf(1) {
		return math.Inf(1)
	}
	balanceScore := GetBalanceScore(metadata, candidate)
	if balanceScore == math.Inf(1) {
		return math.Inf(1)
	}

	// log.Log(log.Custom).Info(fmt.Sprintf("effect score is %v", effectScore))

	// effectScore: 0 ~ 100
	// fairnessScore: 1/len(user) ~ 1
	// lenUser: len(user)
	
	if fairnessScore >= 0.9 {
		fairnessScore = 1
	}

	score := effectScore / fairnessScore

	scoreMin := 0.0
	scoreMax := 100.0 * float64(metadata.GetUserCount())

	// 正規化 score 到 0 ~ 100
	normalizedScore := (score - scoreMin) / (scoreMax - scoreMin) * 100

	// 保證正規化分數在 0 ~ 100 範圍內
	if normalizedScore < 0 {
	    normalizedScore = 0
	} else if normalizedScore > 100 {
	    normalizedScore = 100
	}

	// log.Log(log.Custom).Info(fmt.Sprintf("score is %v", normalizedScore))
	return normalizedScore
}