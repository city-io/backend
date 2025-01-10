package state

import (
	"sync"

	"github.com/asynkron/protoactor-go/actor"
)

type PIDManager struct {
	UserPIDs map[string]*actor.PID

	UserMutex sync.RWMutex
}

var _pm *PIDManager
var once sync.Once

func initPM() {
	_pm = &PIDManager{
		UserPIDs: make(map[string]*actor.PID),
	}
}

func getPM() *PIDManager {
	once.Do(initPM)
	return _pm
}

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
