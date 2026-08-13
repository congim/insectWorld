// Package gamepack 通用游戏包模块测试，验证真实游戏包和失败边界。
package gamepack

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testEngineVersion = "0.1.0"

type manifestErrorCase struct {
	name   string          // 子测试名称
	mutate func(*Manifest) // 对合法manifest施加的单一破坏
	target error           // 期望的稳定错误类型
}

type compiledErrorCase struct {
	name   string              // 子测试名称
	mutate func(*CompiledPack) // 对合法编译结果施加的单一破坏
	target error               // 期望的稳定错误类型
}

// TestLoadAndCompileRealPacks 验证两个题材包共用同一契约且无需修改通用内核。
func TestLoadAndCompileRealPacks(t *testing.T) {
	repositoryRoot := filepath.Clean(filepath.Join("..", "..", "..", ".."))
	tests := []string{"insect-world", "frontier-demo"}
	for _, packID := range tests {
		t.Run(packID, func(t *testing.T) {
			pack, err := LoadAndCompile(filepath.Join(repositoryRoot, "gamepacks", packID), testEngineVersion)
			require.NoError(t, err)
			assert.Equal(t, packID, pack.Manifest.ID)
			assert.Equal(t, packID, pack.Game.ID)
			assert.Len(t, pack.Factions, 1)
			assert.Len(t, pack.Resources, 1)
			assert.Len(t, pack.Units, 1)
			assert.Len(t, pack.Buildings, 1)
			assert.Len(t, pack.Terrains, 1)
			assert.Len(t, pack.Maps, 1)
		})
	}
}

// TestLoadAndCompileRejectsUnknownManifestField 验证manifest未知字段能定位并拒绝。
func TestLoadAndCompileRejectsUnknownManifestField(t *testing.T) {
	root := t.TempDir()
	content := "api_version: 1\nid: invalid-pack\nname: 无效包\nversion: 0.1.0\nunknown_field: true\n"
	require.NoError(t, os.WriteFile(filepath.Join(root, manifestFileName), []byte(content), 0o600))

	_, err := LoadAndCompile(root, testEngineVersion)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrManifestInvalid)
	assert.Contains(t, err.Error(), "unknown_field")
}

// TestDecodeJSONRejectsUnknownConfigField 验证配置未知字段返回具体文件和字段名。
func TestDecodeJSONRejectsUnknownConfigField(t *testing.T) {
	path := filepath.Join(t.TempDir(), "game.json")
	content := `{"id":"test-pack","default_faction_id":"settlers","default_map_id":"plain-map","unknown_field":true}`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	var game GameConfig
	err := decodeJSON(path, &game)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrFileInvalid)
	assert.Contains(t, err.Error(), "game.json")
	assert.Contains(t, err.Error(), "unknown_field")
}

// TestDecodeJSONRejectsMultipleValues 验证单个配置文件不能拼接多个JSON值。
func TestDecodeJSONRejectsMultipleValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "game.json")
	require.NoError(t, os.WriteFile(path, []byte(`{} {}`), 0o600))

	var game GameConfig
	err := decodeJSON(path, &game)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrFileInvalid)
	assert.Contains(t, err.Error(), "只能包含一个JSON值")
}

// TestDecodeJSONRejectsMissingFile 验证缺失配置文件返回文件级错误。
func TestDecodeJSONRejectsMissingFile(t *testing.T) {
	var game GameConfig
	err := decodeJSON(filepath.Join(t.TempDir(), "missing.json"), &game)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrFileInvalid)
}

// TestLoadManifestRejectsMultipleDocuments 验证manifest不能包含多个YAML文档。
func TestLoadManifestRejectsMultipleDocuments(t *testing.T) {
	path := filepath.Join(t.TempDir(), manifestFileName)
	require.NoError(t, os.WriteFile(path, []byte("api_version: 1\n---\napi_version: 1\n"), 0o600))

	_, err := loadManifest(path)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrManifestInvalid)
	assert.Contains(t, err.Error(), "只能包含一个YAML文档")
}

// TestValidateManifestRejectsEscapedPath 验证配置路径不能越出游戏包目录。
func TestValidateManifestRejectsEscapedPath(t *testing.T) {
	manifest := validTestManifest()
	manifest.Configs[0].Path = "../game.json"

	err := validateManifest(t.TempDir(), manifest, testEngineVersion)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrFileInvalid)
	assert.Contains(t, err.Error(), "越出游戏包目录")
}

// TestValidateManifestRejectsEngineMismatch 验证引擎版本不兼容时拒绝加载。
func TestValidateManifestRejectsEngineMismatch(t *testing.T) {
	err := validateManifest(t.TempDir(), validTestManifest(), "0.2.0")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrEngineMismatch)
}

// TestValidateManifestRejectsInvalidDeclarations 覆盖manifest各类结构与兼容性错误。
func TestValidateManifestRejectsInvalidDeclarations(t *testing.T) {
	tests := []manifestErrorCase{
		{name: "契约版本", mutate: func(value *Manifest) { value.APIVersion = 2 }, target: ErrManifestInvalid},
		{name: "稳定ID", mutate: func(value *Manifest) { value.ID = "Invalid_ID" }, target: ErrIDInvalid},
		{name: "展示名称", mutate: func(value *Manifest) { value.Name = " " }, target: ErrManifestInvalid},
		{name: "游戏包版本", mutate: func(value *Manifest) { value.Version = "1.0" }, target: ErrManifestInvalid},
		{name: "未知模块", mutate: func(value *Manifest) { value.Modules[0] = "unknown" }, target: ErrManifestInvalid},
		{name: "重复模块", mutate: func(value *Manifest) { value.Modules[1] = value.Modules[0] }, target: ErrManifestInvalid},
		{name: "缺少模块", mutate: func(value *Manifest) { value.Modules = value.Modules[:3] }, target: ErrManifestInvalid},
		{name: "空模块", mutate: func(value *Manifest) { value.Modules = nil }, target: ErrManifestInvalid},
		{name: "未知配置", mutate: func(value *Manifest) { value.Configs[0].ID = "unknown" }, target: ErrManifestInvalid},
		{name: "重复配置", mutate: func(value *Manifest) { value.Configs = append(value.Configs, value.Configs[0]) }, target: ErrManifestInvalid},
		{name: "schema不匹配", mutate: func(value *Manifest) { value.Configs[0].Schema = "game/v2" }, target: ErrManifestInvalid},
		{name: "非必需配置", mutate: func(value *Manifest) { value.Configs[0].Required = false }, target: ErrManifestInvalid},
		{name: "配置扩展名", mutate: func(value *Manifest) { value.Configs[0].Path = "configs/game.yaml" }, target: ErrManifestInvalid},
		{name: "配置路径不规范", mutate: func(value *Manifest) { value.Configs[0].Path = "other/game.json" }, target: ErrManifestInvalid},
		{name: "缺少配置", mutate: func(value *Manifest) { value.Configs = value.Configs[:len(value.Configs)-1] }, target: ErrManifestInvalid},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manifest := validTestManifest()
			test.mutate(&manifest)
			err := validateManifest(t.TempDir(), manifest, testEngineVersion)
			require.Error(t, err)
			assert.ErrorIs(t, err, test.target)
		})
	}
}

// TestValidateCompiledPackRejectsBrokenReference 验证断裂引用返回字段级错误。
func TestValidateCompiledPackRejectsBrokenReference(t *testing.T) {
	pack := validCompiledPack()
	pack.Units[0].FactionID = "missing-faction"

	err := validateCompiledPack(pack)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrReferenceBroken))
	assert.Contains(t, err.Error(), "configs/units.json.items[0].faction_id")
}

// TestValidateCompiledPackRejectsDuplicateID 验证同类稳定ID重复时拒绝编译。
func TestValidateCompiledPackRejectsDuplicateID(t *testing.T) {
	pack := validCompiledPack()
	pack.Resources = append(pack.Resources, pack.Resources[0])

	err := validateCompiledPack(pack)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrIDInvalid)
	assert.Contains(t, err.Error(), "重复")
}

// TestValidateCompiledPackRejectsInvalidContent 覆盖内容数值、空集合和跨文档引用错误。
func TestValidateCompiledPackRejectsInvalidContent(t *testing.T) {
	tests := []compiledErrorCase{
		{name: "游戏ID不一致", mutate: func(value *CompiledPack) { value.Game.ID = "other-pack" }, target: ErrReferenceBroken},
		{name: "默认阵营断裂", mutate: func(value *CompiledPack) { value.Game.DefaultFactionID = "missing" }, target: ErrReferenceBroken},
		{name: "默认地图断裂", mutate: func(value *CompiledPack) { value.Game.DefaultMapID = "missing" }, target: ErrReferenceBroken},
		{name: "资源为空", mutate: func(value *CompiledPack) { value.Resources = nil }, target: ErrFileInvalid},
		{name: "阵营为空", mutate: func(value *CompiledPack) { value.Factions = nil }, target: ErrFileInvalid},
		{name: "单位为空", mutate: func(value *CompiledPack) { value.Units = nil }, target: ErrFileInvalid},
		{name: "建筑为空", mutate: func(value *CompiledPack) { value.Buildings = nil }, target: ErrFileInvalid},
		{name: "地形为空", mutate: func(value *CompiledPack) { value.Terrains = nil }, target: ErrFileInvalid},
		{name: "地图为空", mutate: func(value *CompiledPack) { value.Maps = nil }, target: ErrFileInvalid},
		{name: "阵营名称", mutate: func(value *CompiledPack) { value.Factions[0].Name = "" }, target: ErrFileInvalid},
		{name: "初始资源引用", mutate: func(value *CompiledPack) { value.Factions[0].StartingResources = map[string]int64{"missing": 1} }, target: ErrReferenceBroken},
		{name: "初始资源负数", mutate: func(value *CompiledPack) { value.Factions[0].StartingResources["supply"] = -1 }, target: ErrFileInvalid},
		{name: "资源名称", mutate: func(value *CompiledPack) { value.Resources[0].Name = "" }, target: ErrFileInvalid},
		{name: "单位名称", mutate: func(value *CompiledPack) { value.Units[0].Name = "" }, target: ErrFileInvalid},
		{name: "单位阵营", mutate: func(value *CompiledPack) { value.Units[0].FactionID = "missing" }, target: ErrReferenceBroken},
		{name: "单位战斗值", mutate: func(value *CompiledPack) { value.Units[0].Attack = 0 }, target: ErrFileInvalid},
		{name: "训练资源引用", mutate: func(value *CompiledPack) { value.Units[0].TrainCost = map[string]int64{"missing": 1} }, target: ErrReferenceBroken},
		{name: "训练成本", mutate: func(value *CompiledPack) { value.Units[0].TrainCost["supply"] = 0 }, target: ErrFileInvalid},
		{name: "建筑名称", mutate: func(value *CompiledPack) { value.Buildings[0].Name = "" }, target: ErrFileInvalid},
		{name: "建筑阵营", mutate: func(value *CompiledPack) { value.Buildings[0].FactionID = "missing" }, target: ErrReferenceBroken},
		{name: "建造时长", mutate: func(value *CompiledPack) { value.Buildings[0].BuildTimeMs = 0 }, target: ErrFileInvalid},
		{name: "建造成本", mutate: func(value *CompiledPack) { value.Buildings[0].BuildCost = nil }, target: ErrFileInvalid},
		{name: "可训练单位为空", mutate: func(value *CompiledPack) { value.Buildings[0].TrainableUnitIDs = nil }, target: ErrFileInvalid},
		{name: "可训练单位引用", mutate: func(value *CompiledPack) { value.Buildings[0].TrainableUnitIDs[0] = "missing" }, target: ErrReferenceBroken},
		{name: "地形名称", mutate: func(value *CompiledPack) { value.Terrains[0].Name = "" }, target: ErrFileInvalid},
		{name: "移动消耗", mutate: func(value *CompiledPack) { value.Terrains[0].MoveCost = 0 }, target: ErrFileInvalid},
		{name: "地图名称", mutate: func(value *CompiledPack) { value.Maps[0].Name = "" }, target: ErrFileInvalid},
		{name: "地图尺寸", mutate: func(value *CompiledPack) { value.Maps[0].Width = 0 }, target: ErrFileInvalid},
		{name: "默认地形引用", mutate: func(value *CompiledPack) { value.Maps[0].DefaultTerrainID = "missing" }, target: ErrReferenceBroken},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pack := validCompiledPack()
			test.mutate(pack)
			err := validateCompiledPack(pack)
			require.Error(t, err)
			assert.ErrorIs(t, err, test.target)
		})
	}
}

// TestVersionValidation 覆盖语义版本格式和兼容区间边界。
func TestVersionValidation(t *testing.T) {
	_, err := parseVersion("01.0.0")
	require.Error(t, err)
	_, err = parseVersion("a.0.0")
	require.Error(t, err)

	assert.ErrorIs(t, validateEngineConstraint(EngineConstraint{MinVersion: "bad", MaxExclusive: "0.2.0"}, testEngineVersion), ErrManifestInvalid)
	assert.ErrorIs(t, validateEngineConstraint(EngineConstraint{MinVersion: "0.1.0", MaxExclusive: "bad"}, testEngineVersion), ErrManifestInvalid)
	assert.ErrorIs(t, validateEngineConstraint(EngineConstraint{MinVersion: "0.2.0", MaxExclusive: "0.1.0"}, testEngineVersion), ErrManifestInvalid)
	assert.ErrorIs(t, validateEngineConstraint(EngineConstraint{MinVersion: "0.1.0", MaxExclusive: "0.2.0"}, "bad"), ErrEngineMismatch)
	assert.ErrorIs(t, validateEngineConstraint(EngineConstraint{MinVersion: "0.1.0", MaxExclusive: "0.2.0"}, "0.0.9"), ErrEngineMismatch)
}

func validTestManifest() Manifest {
	configs := make([]ConfigFile, 0, len(expectedSchemas))
	orderedIDs := []string{ConfigIDGame, ConfigIDFactions, ConfigIDResources, ConfigIDUnits, ConfigIDBuildings, ConfigIDTerrains, ConfigIDMaps}
	for _, id := range orderedIDs {
		configs = append(configs, ConfigFile{ID: id, Path: "configs/" + id + ".json", Schema: expectedSchemas[id], Required: true})
	}
	return Manifest{
		APIVersion: ManifestAPIVersionV1,
		ID:         "test-pack",
		Name:       "测试包",
		Version:    "0.1.0",
		Engine:     EngineConstraint{MinVersion: "0.1.0", MaxExclusive: "0.2.0"},
		Modules:    []string{"world", "growth", "combat", "economy"},
		Configs:    configs,
	}
}

func validCompiledPack() *CompiledPack {
	return &CompiledPack{
		Manifest:  validTestManifest(),
		Game:      GameConfig{ID: "test-pack", DefaultFactionID: "settlers", DefaultMapID: "plain-map"},
		Factions:  []FactionConfig{{ID: "settlers", Name: "拓荒者", StartingResources: map[string]int64{"supply": 100}}},
		Resources: []ResourceConfig{{ID: "supply", Name: "补给"}},
		Units: []UnitConfig{{
			ID: "scout", Name: "侦察队", FactionID: "settlers", Attack: 1, Defense: 1, Health: 1,
			TrainCost: map[string]int64{"supply": 1}, TrainTimeMs: 1,
		}},
		Buildings: []BuildingConfig{{
			ID: "outpost", Name: "前哨站", FactionID: "settlers", BuildCost: map[string]int64{"supply": 1},
			BuildTimeMs: 1, TrainableUnitIDs: []string{"scout"},
		}},
		Terrains: []TerrainConfig{{ID: "plain", Name: "平地", MoveCost: 1}},
		Maps:     []MapConfig{{ID: "plain-map", Name: "平原", Width: 1, Height: 1, DefaultTerrainID: "plain"}},
	}
}
