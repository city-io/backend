package battles

import (
	"sort"
	"sync"

	"cityio/internal/domain"
)

var registry = struct {
	sync.RWMutex
	items map[string]domain.Battle
}{items: make(map[string]domain.Battle)}

func Upsert(battle domain.Battle) {
	registry.Lock()
	registry.items[battle.BattleID] = battle
	registry.Unlock()
}

func Delete(id string) {
	registry.Lock()
	delete(registry.items, id)
	registry.Unlock()
}

func Get(id string) (domain.Battle, bool) {
	registry.RLock()
	battle, ok := registry.items[id]
	registry.RUnlock()
	return battle, ok
}

func All() []domain.Battle {
	registry.RLock()
	result := make([]domain.Battle, 0, len(registry.items))
	for _, battle := range registry.items {
		result = append(result, battle)
	}
	registry.RUnlock()
	sort.Slice(result, func(i, j int) bool { return result[i].BattleID < result[j].BattleID })
	return result
}
