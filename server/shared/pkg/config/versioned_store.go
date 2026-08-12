// Package config 共享内核配置模块，提供配置加载/校验/查询统一API。
// 本文件实现版本化配置存储（ADR-004 3.1）：保留历史版本配置供快照类业务查询，
// 版本保留受引用计数保护——进行中实例引用的版本不被历史滚动清理（ADR-004 3.4 双向回滚）。
package config

import (
	"fmt"
	"sort"
	"sync"
)

// maxConfigVersionHistory 保留的最大历史配置版本数（默认10，ADR-004 3.1）。
// 超出上限且引用计数为0的版本允许清理；有引用计数的版本永不清理。
const maxConfigVersionHistory = 10

// VersionedConfigStore 版本化配置存储，维护"版本号→扩展点ID→配置项ID→配置值"的映射。
// 配置热更编译时逐条目写入；快照类业务（战斗/生产队列）创建时Pin、结束时Unpin，
// 进行中实例引用的版本永不因历史滚动被清理。
type VersionedConfigStore struct {
	mu       sync.RWMutex                        // 读写锁，保护版本映射与引用计数并发安全
	versions map[int64]map[string]map[string]any // 版本号 → 扩展点ID → 配置项ID → 配置值
	refs     map[int64]int64                     // 版本号 → 引用计数，>0的版本禁止清理
}

// NewVersionedConfigStore 创建版本化配置存储实例。
func NewVersionedConfigStore() *VersionedConfigStore {
	return &VersionedConfigStore{
		versions: make(map[int64]map[string]map[string]any),
		refs:     make(map[int64]int64),
	}
}

// PutEntry 写入单个配置项到指定版本。
// version为配置版本号，extPointID为扩展点ID，key为配置项ID，value为配置值。
// 配置加载/热更编译时调用，同一版本重复写入同key覆盖旧值（最后一次编译结果生效）。
func (s *VersionedConfigStore) PutEntry(version int64, extPointID string, key string, value any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.versions[version] == nil {
		s.versions[version] = make(map[string]map[string]any)
	}
	if s.versions[version][extPointID] == nil {
		s.versions[version][extPointID] = make(map[string]any)
	}
	s.versions[version][extPointID][key] = value
}

// PutExtPoint 写入整个扩展点配置到指定版本（配置无法拆分为条目的退化路径）。
// version为配置版本号，extPointID为扩展点ID，value为扩展点整体配置值，
// 以扩展点ID作为配置项key存储，GetWithVersion按同一key可查回。
func (s *VersionedConfigStore) PutExtPoint(version int64, extPointID string, value any) {
	s.PutEntry(version, extPointID, extPointID, value)
}

// Get 按指定版本查询配置项。
// 版本不可用返回ErrConfigVersionGone；版本存在但配置项缺失返回nil（存在性用Has判断）。
func (s *VersionedConfigStore) Get(version int64, extPointID string, key string) (any, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	extPoints, ok := s.versions[version]
	if !ok {
		return nil, fmt.Errorf("配置版本 %d 不可用: %w", version, ErrConfigVersionGone)
	}
	items, ok := extPoints[extPointID]
	if !ok {
		return nil, nil
	}
	return items[key], nil
}

// Has 判断指定版本中配置项是否存在。
// 版本不可用返回ErrConfigVersionGone；配置项存在返回true，否则false。
func (s *VersionedConfigStore) Has(version int64, extPointID string, key string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	extPoints, ok := s.versions[version]
	if !ok {
		return false, fmt.Errorf("配置版本 %d 不可用: %w", version, ErrConfigVersionGone)
	}
	items, ok := extPoints[extPointID]
	if !ok {
		return false, nil
	}
	_, ok = items[key]
	return ok, nil
}

// Pin 锁定配置版本，引用计数+1；快照类业务创建时调用（ADR-004 3.1版本保留机制）。
// 版本不可用返回ErrConfigVersionGone。
func (s *VersionedConfigStore) Pin(version int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.versions[version]; !ok {
		return fmt.Errorf("配置版本 %d 不可用: %w", version, ErrConfigVersionGone)
	}
	s.refs[version]++
	return nil
}

// Unpin 释放配置版本引用，引用计数-1（不低于0）；业务结束时调用。
// 归零且超出保留上限后由PruneUnreferenced清理（ADR-004 3.4）。
// 版本不可用返回ErrConfigVersionGone。
func (s *VersionedConfigStore) Unpin(version int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.versions[version]; !ok {
		return fmt.Errorf("配置版本 %d 不可用: %w", version, ErrConfigVersionGone)
	}
	if s.refs[version] > 0 {
		s.refs[version]--
	}
	return nil
}

// PinCount 返回指定版本的引用计数（观测/测试用）。
func (s *VersionedConfigStore) PinCount(version int64) int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.refs[version]
}

// HasVersion 判断指定版本是否存在于存储（回滚目标校验/观测用，ADR-004 3.4）。
// 回滚命令在切换当前版本前校验目标版本存在，保证回滚原子性（校验失败保持当前版本不变）。
func (s *VersionedConfigStore) HasVersion(version int64) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.versions[version]
	return ok
}

// VersionCount 返回已记录版本数（观测/测试用）。
func (s *VersionedConfigStore) VersionCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.versions)
}

// PruneUnreferenced 清理超出保留上限且引用计数为0的历史版本，返回被清理的版本号列表。
// 有引用计数的版本永不清理（ADR-004 3.4：禁止在存在引用时物理删除配置版本）。
func (s *VersionedConfigStore) PruneUnreferenced() []int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	var candidates []int64
	for v := range s.versions {
		if s.refs[v] == 0 {
			candidates = append(candidates, v)
		}
	}
	if len(candidates) <= maxConfigVersionHistory {
		return nil
	}
	// 未引用版本按版本号升序，保留最新的maxConfigVersionHistory个，更老的清理
	sort.Slice(candidates, func(i, j int) bool { return candidates[i] < candidates[j] })
	toRemove := candidates[:len(candidates)-maxConfigVersionHistory]
	removed := make([]int64, 0, len(toRemove))
	for _, v := range toRemove {
		delete(s.versions, v)
		delete(s.refs, v)
		removed = append(removed, v)
	}
	return removed
}
