package state

import (
	"sync"

	"github.com/asynkron/protoactor-go/actor"
)

type PIDManager struct {
	UserPIDs map[string]*actor.PID
	CityPIDs map[string]*actor.PID

	UserMutex sync.RWMutex
	CityMutex sync.RWMutex
}

var _pm *PIDManager
var once sync.Once

func initPM() {
	_pm = &PIDManager{
		UserPIDs: make(map[string]*actor.PID),
		CityPIDs: make(map[string]*actor.PID),
	}
}

func getPM() *PIDManager {
	once.Do(initPM)
	return _pm
}

// Users
func AddUserPID(userId string, pid *actor.PID) {
	pm := getPM()
	pm.UserMutex.Lock()
	defer pm.UserMutex.Unlock()
	pm.UserPIDs[userId] = pid
}

func GetUserPID(userId string) (*actor.PID, bool) {
	pm := getPM()
	pm.UserMutex.RLock()
	defer pm.UserMutex.RUnlock()
	pid, exists := pm.UserPIDs[userId]
	return pid, exists
}

func RemoveUserPID(userId string) {
	pm := getPM()
	pm.UserMutex.Lock()
	defer pm.UserMutex.Unlock()
	delete(pm.UserPIDs, userId)
}

// Cities
func AddCityPID(cityId string, pid *actor.PID) {
	pm := getPM()
	pm.CityMutex.Lock()
	defer pm.CityMutex.Unlock()
	pm.CityPIDs[cityId] = pid
}

func GetCityPID(cityId string) (*actor.PID, bool) {
	pm := getPM()
	pm.CityMutex.RLock()
	defer pm.CityMutex.RUnlock()
	pid, exists := pm.CityPIDs[cityId]
	return pid, exists
}

func RemoveCityPID(cityId string) {
	pm := getPM()
	pm.CityMutex.Lock()
	defer pm.CityMutex.Unlock()
	delete(pm.CityPIDs, cityId)
}
