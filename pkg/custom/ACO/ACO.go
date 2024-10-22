package ACO

import (
	"container/heap"
	"math"
	"math/rand"
	"sync"
	"time"

	// "fmt"

	Metadata "github.com/apache/yunikorn-core/pkg/custom/metadata"

	ACOHeap "github.com/apache/yunikorn-core/pkg/custom/ACO/heap"
	ACOmath "github.com/apache/yunikorn-core/pkg/custom/ACO/math"

	AGAmath "github.com/apache/yunikorn-core/pkg/custom/math"
	"github.com/apache/yunikorn-core/pkg/custom/math/vector"
	// "github.com/apache/yunikorn-core/pkg/log"
)

type ACOHyperParameter struct {
	AntNum			int
	Epochs			int
	Steps 			int
}

func NewACOHyperParameter(antNum int, epochs int, steps int) *ACOHyperParameter {
	return &ACOHyperParameter{
		AntNum:        antNum,
		Epochs:        epochs,
		Steps: 		   steps,
	}
}

type ACO struct {
	HyperParameter 	*ACOHyperParameter

	metadata 		*Metadata.Metadata
	pheromone 		*ACOHeap.CoordinateHeap
	candidates		[]*vector.Vector

	sync.RWMutex
}

var aco *ACO

func Init() {
	aco = NewACO()
}

func NewACO() *ACO {
	return &ACO{}
}

func GetACO() *ACO {
	return aco
}

func (aco *ACO) SetHyperParameter(parameters *ACOHyperParameter) {
	aco.HyperParameter = parameters
}

func (aco *ACO) SetMetaData(metadata *Metadata.Metadata) {
	aco.metadata = metadata
}

// 根據 neighbors 的 value 作為權重隨機選擇下一個座標
func selectNextCoordinate(neighbors []*ACOHeap.CoordinateInfo) []float64 {
	if len(neighbors) == 0 {
		return nil
	}

	// 根據 value 作為機率分佈進行選擇
	totalValue := 0.0
	for _, neighbor := range neighbors {
		totalValue += neighbor.Value
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano())).Float64() * totalValue

	for _, neighbor := range neighbors {
		r -= neighbor.Value
		if r <= 0 {
			return neighbor.Coordinates
		}
	}
	return neighbors[len(neighbors)-1].Coordinates
}

// 查找與目標距離為 1 的座標
func findNeighbors(target []float64, h ACOHeap.CoordinateHeap) []*ACOHeap.CoordinateInfo {
	var neighbors []*ACOHeap.CoordinateInfo

	// 找出曼哈頓距離為 1 的座標
	for _, coord := range h {
		if ACOmath.ManhattanDistance(coord.Coordinates, target) == 1 {
			neighbors = append(neighbors, coord)
		}
	}

	// 如果堆中沒有相關的鄰居，創建新的鄰居座標
	if len(neighbors) == 0 {
		dimensions := len(target)
		for i := 0; i < dimensions; i++ {
			neighborCoord := make([]float64, dimensions)
			copy(neighborCoord, target)
			neighborCoord[i] += 1 // 產生相距 1 的座標
			neighbors = append(neighbors, &ACOHeap.CoordinateInfo{
				Coordinates: neighborCoord,
				Value:       1.0, // 預設值
			})

			if target[i] == 0 {
				continue
			}
			neighborCoordNegative := make([]float64, dimensions)
			copy(neighborCoordNegative, target)
			neighborCoordNegative[i] -= 1 // 相反方向
			neighbors = append(neighbors, &ACOHeap.CoordinateInfo{
				Coordinates: neighborCoordNegative,
				Value:       1.0, // 預設值
			})
		}
	}

	return neighbors
}

func (aco *ACO) Start(candidates []*vector.Vector) {
//ACO hyper parameter
	numAnts := aco.HyperParameter.AntNum
	epochs := aco.HyperParameter.Epochs
	steps := aco.HyperParameter.Steps

	// users := aco.metadata.UserData.UserCount
	// nodes := aco.metadata.NodeData.NodeCount

	aco.candidates = candidates
	aco.pheromone = &ACOHeap.CoordinateHeap{}
	heap.Init(aco.pheromone)

	for iteration := 0; iteration < epochs; iteration++ {
		// log.Log(log.Custom).Info(fmt.Sprintf("iteration %v start", iteration))
		paths := make([][][]float64, 0)
		scores := make([]float64, 0)
		best_score := math.MaxFloat64

		for ant_index := 0; ant_index < int(numAnts); ant_index++ {
			path := make([][]float64, 0)
		
			current_position := aco.candidates[ant_index]
			path = append(path, current_position.ToArray())

			for step := 0; step < steps; step++ {
				neighbors := findNeighbors(current_position.ToArray(), *aco.pheromone)
				nextPosition := selectNextCoordinate(neighbors)
				current_position = vector.NewVector(nextPosition)
				path = append(path, current_position.ToArray())
			}

			score := AGAmath.GetScore(aco.metadata, current_position)
      		scores = append(scores, score)

			if score < best_score {
      		  	best_score = score
			}
      		paths = append(paths, path)
		}

		// update pheromone
		softmaxScores := ACOmath.SoftmaxNormalize(paths, scores)
		for _, item := range softmaxScores {
			// log.Log(log.Custom).Info(fmt.Sprintf("item is %v", item))
			coordInfo := aco.pheromone.Find(item.Coord)
			// log.Log(log.Custom).Info(fmt.Sprintf("coordInfo is %v", coordInfo))
			if coordInfo == nil {
				heap.Push(aco.pheromone, &ACOHeap.CoordinateInfo{Coordinates: item.Coord, Value: 1})
			}
			coordInfo = aco.pheromone.Find(item.Coord)

			aco.pheromone.Update(coordInfo, coordInfo.Value / math.Exp(item.Score - 0.5))
			// fmt.Printf("Coordinate: %v, Softmax Score: %f\n", item.Coord, item.Score)
		}
	}
	//ACO
}

func (aco *ACO) GetCandidate(amount int) []*vector.Vector{
	returnCandidates := make([]*vector.Vector, 0)
	nums := int(math.Min(float64(amount), float64(aco.pheromone.Len())))
	for i := 0; i < nums; i++ {
		tmpVector := vector.NewVector(aco.pheromone.Pop().(*ACOHeap.CoordinateInfo).Coordinates)
		returnCandidates = append(returnCandidates, tmpVector)
	}
	return returnCandidates
}