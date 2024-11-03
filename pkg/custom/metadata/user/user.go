package UserData

import (
	"sync"
	"github.com/apache/yunikorn-core/pkg/scheduler/objects"
)

type UserData struct {
	resourceTypes  	[]string

	userCount      	int
	userNames		[]string
	userAskMap 		map[string]*AskData

	sync.RWMutex
}

type AskData struct {
	askCount		int
	userAsks	 	[]float64
	requests		[]*objects.AllocationAsk
}

func NewUserData(resourceTypes []string) *UserData {
	return &UserData{
		userCount:  0,
		userNames: make([]string, 0),
		resourceTypes:   resourceTypes,
		userAskMap : make(map[string]*AskData),
	}
}

func (userData *UserData) GetUserCount() int{
	return userData.userCount
}

func (userData *UserData) UpdateUserInfo(apps []*objects.Application) {
	userData.Lock()
	defer userData.Unlock()

	userData.userCount = len(apps)
	userData.userNames = make([]string, 0)
	userData.userAskMap = make(map[string]*AskData)
	for _, app := range apps {
		userData.userNames = append(userData.userNames, app.ApplicationID)

		requests := app.GetAllRequests()

		if len(requests) == 0 {
			continue
		}

		pendingAsks := 0
		for _, request := range requests {
			pendingAsks += int(request.GetPendingAskRepeat())
		}

		askData := &AskData{
			askCount: 	pendingAsks,
			userAsks:	userData.praseAskResources(requests[0]),
			requests:	requests,
		}

		_, exist := userData.userAskMap[app.ApplicationID]
		if !exist {
			userData.userAskMap[app.ApplicationID] = askData
		}
	}
}

func (userData *UserData) praseAskResources(ask *objects.AllocationAsk) []float64{
	userAsk := make([]float64, len(userData.resourceTypes))	

	curResource := ask.GetAllocatedResource().Resources
	for index, targetType := range userData.resourceTypes {
		userAsk[index] += float64(curResource[targetType])	
	}
	
	return userAsk
}

func (userData *UserData) GetUserAsks() [][]float64{
	asks := make([][]float64, 0)
	for _, appName := range userData.userNames {
		asks = append(asks, userData.userAskMap[appName].userAsks)
	}
	return asks
}

func (userData *UserData) GetUserAskCount(index int) int{
	userName := userData.userNames[index]
	return userData.userAskMap[userName].askCount
}

func (userData *UserData) GetLastRequest(index int) *objects.AllocationAsk {
	userName := userData.userNames[index]
	totalNum := userData.userAskMap[userName].askCount
	return userData.userAskMap[userName].requests[totalNum-1]
}