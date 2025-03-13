package GOA

import (
	"math"
	"sync"

	goamath "github.com/apache/yunikorn-core/pkg/custom/GOA/math"
	agamath "github.com/apache/yunikorn-core/pkg/custom/math"
	"github.com/apache/yunikorn-core/pkg/custom/math/vector"
	Metadata "github.com/apache/yunikorn-core/pkg/custom/metadata"
	"github.com/apache/yunikorn-core/pkg/metrics"
)


type GOAHyperParameter struct {
	iterations        int
	cMax              float64
	cMin              float64
	GrasshopperAmount int
	GForce            float64
	WindForce         float64
}

func NewGOAHyperParameter(iterations int, cMax float64, cMin float64, grasshopperAmount int, GForce float64, WindForce float64) *GOAHyperParameter {
	return &GOAHyperParameter{
		iterations:        iterations,
		cMax:              cMax,
		cMin:              cMin,
		GrasshopperAmount: grasshopperAmount,
		GForce:            GForce,
		WindForce:         WindForce,
	}
}

type GOA struct {
	metadata        *Metadata.Metadata
	fairness_vector *vector.Vector

	grasshoppers    []*vector.Vector
	bestGrasshopper *vector.Vector

	hyperParameter *GOAHyperParameter

	sync.RWMutex
}

func (goa *GOA) SetHyperParameter(hyperParameter *GOAHyperParameter) {
	goa.hyperParameter = hyperParameter
	goa.grasshoppers =  make([]*vector.Vector, goa.hyperParameter.GrasshopperAmount)
}

func(goa *GOA) SetMetaData(metaData *Metadata.Metadata){
	goa.metadata = metaData
}

func NewGOA() *GOA {
	return &GOA{}
}

// GOA utils

func (goa *GOA) getGravityUnitVector(grasshopper *vector.Vector) *vector.Vector {
	grasshopper_unit := grasshopper.GetUnitVector()

	t := goa.fairness_vector.Dot(grasshopper_unit) / grasshopper_unit.Dot(grasshopper_unit)

	D := vector.Multiple(grasshopper_unit, t)

	gravity_vector := vector.Subtract(goa.fairness_vector, D)

	return gravity_vector.GetUnitVector()
}

func (goa *GOA) getWindUnitVector(grasshopper *vector.Vector, best_grasshopper *vector.Vector) *vector.Vector {
	wind_vector := vector.Subtract(best_grasshopper, grasshopper)
	return wind_vector.GetUnitVector()
}

func (goa *GOA) greedyMove(minValue float64, nextPositions *[]*vector.Vector) {
	// goa.bestGrasshopper = nil

	for i := 0; i < goa.hyperParameter.GrasshopperAmount; i++ {
		nextPosition := (*nextPositions)[i]

		oldValue := agamath.GetScore(goa.metadata, goa.grasshoppers[i])
		newValue := agamath.GetScore(goa.metadata, nextPosition)

		if newValue != math.Inf(1) {
			if newValue < oldValue {

				goa.grasshoppers[i] = nextPosition
				oldValue = newValue
			}

			if oldValue < minValue || goa.bestGrasshopper == nil {
				minValue = oldValue

				goa.bestGrasshopper = goa.grasshoppers[i]
			}
		}

		// 加入 G_force * calculate_gravity_unit_vector 的結果到 grasshoppers[i]
		gravityVector := goa.getGravityUnitVector(goa.grasshoppers[i])
		goa.grasshoppers[i].Add(vector.Multiple(gravityVector, goa.hyperParameter.GForce))

	}
}

func (goa *GOA) calculateDomainResources() {
	goa.metadata.CalculateDRs()
	fairness_array := make([]float64, 0)
	for _, value := range goa.metadata.DRRatioReciprocals {
		fairness_array = append(fairness_array, value...)
	}
	goa.fairness_vector = vector.NewVector(fairness_array)
}


func (goa *GOA) Start(candidates []*vector.Vector) (decision []int) {
	users := goa.metadata.GetUserCount()
	nodes := goa.metadata.GetNodeCount()

	if users*nodes == 0 {
		return nil
	}

	goa.calculateDomainResources()

	minValue := math.Inf(1)
	goa.grasshoppers = candidates
	goa.bestGrasshopper = vector.WithSize(users * nodes)

	scoreSum := 0.0
	scoreCount := 0

	for _, candidate := range(candidates) {
		value := agamath.GetScore(goa.metadata, candidate)
		if value != math.Inf(1) {
			scoreSum += value
			scoreCount += 1
		}
		if minValue > value {
			minValue = value
			goa.bestGrasshopper = candidate
		}
	}

	avgScore := 0.0

	if minValue == math.Inf(1) {
		avgScore = -1
	} else {
		avgScore = scoreSum / float64(scoreCount)
	}

	metrics.GetCustomMetrics().SetInitialCandidateAvgScore(avgScore)

	// main program
	for i := 0; i < goa.hyperParameter.iterations; i++ {
		c := goa.hyperParameter.cMax - float64(i)*((goa.hyperParameter.cMax-goa.hyperParameter.cMin)/float64(goa.hyperParameter.iterations))
		nextPositions := make([]*vector.Vector, 0)
		for grasshopper_i := 0; grasshopper_i < goa.hyperParameter.GrasshopperAmount; grasshopper_i++ {
			moveVector := vector.WithSize(users * nodes)
			for grasshopper_j := 0; grasshopper_j < goa.hyperParameter.GrasshopperAmount; grasshopper_j++ {
				if grasshopper_i == grasshopper_j {
					continue
				}
				distance := vector.Subtract(goa.grasshoppers[grasshopper_i], goa.grasshoppers[grasshopper_j])
				if distance.Norm() == 0 {
					continue
				}
				moveVector.Add(vector.Multiple(distance.GetUnitVector(), goamath.SocialInfluence(distance.Norm())))

			}
			moveVector = vector.Multiple(moveVector, c)

			wind_unit_vector := goa.getWindUnitVector(goa.grasshoppers[grasshopper_i], goa.bestGrasshopper)
			moveVector.Add(vector.Multiple(wind_unit_vector, goa.hyperParameter.WindForce))
			nextPositions = append(nextPositions, vector.Add(goa.grasshoppers[grasshopper_i], moveVector))
		}
		goa.greedyMove(math.Inf(1), &nextPositions)
	}

	decision = goa.bestGrasshopper.ToIntArray()
	return
}
