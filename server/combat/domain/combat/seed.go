// Package combat 战斗聚合根，维护战斗状态与轮次执行。
// 本文件实现战斗轮次随机种子派生（ADR-001 3.3 / ADR-004 3.5），
// config_version取快照冻结值，保证热更/回滚不使重放种子漂移。
package combat

import (
	"fmt"
	"hash/fnv"
)

// DeriveRoundSeed 按快照冻结版本派生轮次随机种子（ADR-001 3.3 / ADR-004 3.5）。
// configVersion必须取CombatSnapshot.configVersion（开战冻结值），禁止取当前热更版本，
// 否则热更/回滚会使重放种子漂移，破坏战斗回放一致性；同combatID同configVersion同round必然同种子。
// 哈希采用FNV-1a（确定性、跨Go版本稳定），不依赖math/rand的版本行为。
func DeriveRoundSeed(combatID int64, configVersion int64, round int) uint64 {
	seed := fnv.New64a()
	seed.Write([]byte(fmt.Sprintf("%d:%d:%d", combatID, configVersion, round)))
	return seed.Sum64()
}
