package state

import (
	"sync"

	"github.com/asynkron/protoactor-go/actor"
)

type PIDManager struct {
	UserPIDs    map[string]*actor.PID
	CityPIDs    map[string]*actor.PID
	MapTilePIDs map[int]map[int]*actor.PID
	ArmyPIDs    map[string]*actor.PID

	UserMutex    sync.RWMutex
	CityMutex    sync.RWMutex
	MapTileMutex sync.RWMutex
}

var _pm *PIDManager
var once sync.Once

func initPM() {
	_pm = &PIDManager{
		UserPIDs:    make(map[string]*actor.PID),
		CityPIDs:    make(map[string]*actor.PID),
		MapTilePIDs: make(map[int]map[int]*actor.PID),
		ArmyPIDs:    make(map[string]*actor.PID),
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

// MapTiles
func AddMapTilePID(x int, y int, pid *actor.PID) {
	pm := getPM()
	pm.MapTileMutex.Lock()
	defer pm.MapTileMutex.Unlock()
	if _, exists := pm.MapTilePIDs[x]; !exists {
		pm.MapTilePIDs[x] = make(map[int]*actor.PID)
	}
	pm.MapTilePIDs[x][y] = pid
}

func GetMapTilePID(x int, y int) (*actor.PID, bool) {
	pm := getPM()
	pm.MapTileMutex.RLock()
	defer pm.MapTileMutex.RUnlock()
	if _, exists := pm.MapTilePIDs[x]; !exists {
		return nil, false
	}
	pid, exists := pm.MapTilePIDs[x][y]
	return pid, exists
}

func RemoveMapTilePID(x int, y int) {
	pm := getPM()
	pm.MapTileMutex.Lock()
	defer pm.MapTileMutex.Unlock()
	if _, exists := pm.MapTilePIDs[x]; exists {
		delete(pm.MapTilePIDs[x], y)
	}
}

// Armies
func AddArmyPID(armyId string, pid *actor.PID) {
	pm := getPM()
	pm.UserMutex.Lock()
	defer pm.UserMutex.Unlock()
	pm.ArmyPIDs[armyId] = pid
}

func GetArmyPID(armyId string) (*actor.PID, bool) {
	pm := getPM()
	pm.UserMutex.RLock()
	defer pm.UserMutex.RUnlock()
	pid, exists := pm.ArmyPIDs[armyId]
	return pid, exists
}

func RemoveArmyPID(armyId string) {
	pm := getPM()
	pm.UserMutex.Lock()
	defer pm.UserMutex.Unlock()
	delete(pm.ArmyPIDs, armyId)
}
