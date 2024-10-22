package AGA

import (
	"sync"

	"github.com/apache/yunikorn-core/pkg/custom/math/vector"
	Metadata "github.com/apache/yunikorn-core/pkg/custom/metadata"
	"github.com/apache/yunikorn-core/pkg/log"

	"github.com/apache/yunikorn-core/pkg/custom/ACO"
	"github.com/apache/yunikorn-core/pkg/custom/GOA"
)

var (
	aga *AGA
) 

type AGA struct {
	metadata 	*Metadata.Metadata
	goa			*GOA.GOA
	aco 		*ACO.ACO

	sync.RWMutex
}

func Init() {
	aga = NewAGA()
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

func (aga *AGA) SetMetaData(newMetadata *Metadata.Metadata) {
	aga.metadata = newMetadata
}

func (aga *AGA) Start(candidates []*vector.Vector, ACOParameter *ACO.ACOHyperParameter, GOAParameter *GOA.GOAHyperParameter) []int{
	aga.aco.SetHyperParameter(ACOParameter)
	aga.aco.SetMetaData(aga.metadata)

	aga.goa.SetHyperParameter(GOAParameter)
	aga.goa.SetMetaData(aga.metadata)

	aga.metadata.CalculateDRs()

	log.Log(log.Custom).Info("AGA start")

	aga.aco.Start(candidates)

	desision := aga.goa.Start(aga.aco.GetCandidate(int(GOAParameter.GrasshopperAmount)))

	return desision
}
