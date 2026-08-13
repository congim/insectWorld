// Package gamepack 通用游戏包模块，负责加载、校验并编译题材内容配置。
package gamepack

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var stableIDPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$`)

var allowedModules = map[string]struct{}{
	"world":   {},
	"growth":  {},
	"combat":  {},
	"economy": {},
}

var expectedSchemas = map[string]string{
	ConfigIDGame:      SchemaGameV1,
	ConfigIDFactions:  SchemaFactionsV1,
	ConfigIDResources: SchemaResourcesV1,
	ConfigIDUnits:     SchemaUnitsV1,
	ConfigIDBuildings: SchemaBuildingsV1,
	ConfigIDTerrains:  SchemaTerrainsV1,
	ConfigIDMaps:      SchemaMapsV1,
}

func validateManifest(root string, manifest Manifest, engineVersion string) error {
	if manifest.APIVersion != ManifestAPIVersionV1 {
		return fmt.Errorf("manifest.api_version=%d，当前仅支持%d: %w", manifest.APIVersion, ManifestAPIVersionV1, ErrManifestInvalid)
	}
	if err := validateStableID("manifest.id", manifest.ID); err != nil {
		return err
	}
	if strings.TrimSpace(manifest.Name) == "" {
		return fmt.Errorf("manifest.name不能为空: %w", ErrManifestInvalid)
	}
	if _, err := parseVersion(manifest.Version); err != nil {
		return fmt.Errorf("manifest.version非法: %w: %w", err, ErrManifestInvalid)
	}
	if err := validateEngineConstraint(manifest.Engine, engineVersion); err != nil {
		return err
	}
	if err := validateModules(manifest.Modules); err != nil {
		return err
	}
	return validateConfigDeclarations(root, manifest.Configs)
}

func validateModules(modules []string) error {
	if len(modules) == 0 {
		return fmt.Errorf("manifest.modules不能为空: %w", ErrManifestInvalid)
	}
	seen := make(map[string]struct{}, len(modules))
	for index, module := range modules {
		if _, ok := allowedModules[module]; !ok {
			return fmt.Errorf("manifest.modules[%d]=%s不受支持: %w", index, module, ErrManifestInvalid)
		}
		if _, ok := seen[module]; ok {
			return fmt.Errorf("manifest.modules[%d]=%s重复: %w", index, module, ErrManifestInvalid)
		}
		seen[module] = struct{}{}
	}
	for module := range allowedModules {
		if _, ok := seen[module]; !ok {
			return fmt.Errorf("首版垂直切片缺少必需模块，module=%s: %w", module, ErrManifestInvalid)
		}
	}
	return nil
}

func validateConfigDeclarations(root string, configs []ConfigFile) error {
	seenIDs := make(map[string]struct{}, len(configs))
	for index, config := range configs {
		expectedSchema, ok := expectedSchemas[config.ID]
		if !ok {
			return fmt.Errorf("manifest.configs[%d].id=%s不受支持: %w", index, config.ID, ErrManifestInvalid)
		}
		if _, ok := seenIDs[config.ID]; ok {
			return fmt.Errorf("manifest.configs[%d].id=%s重复: %w", index, config.ID, ErrManifestInvalid)
		}
		seenIDs[config.ID] = struct{}{}
		if config.Schema != expectedSchema {
			return fmt.Errorf("manifest.configs[%d].schema=%s，应为%s: %w", index, config.Schema, expectedSchema, ErrManifestInvalid)
		}
		if !config.Required {
			return fmt.Errorf("首版配置文档必须标记required，id=%s: %w", config.ID, ErrManifestInvalid)
		}
		if _, err := resolveConfigPath(root, config.Path); err != nil {
			return err
		}
		expectedPath := filepath.ToSlash(filepath.Join("configs", config.ID+".json"))
		if filepath.ToSlash(config.Path) != expectedPath {
			return fmt.Errorf("manifest.configs[%d].path=%s，应为%s: %w", index, config.Path, expectedPath, ErrManifestInvalid)
		}
	}
	for id := range expectedSchemas {
		if _, ok := seenIDs[id]; !ok {
			return fmt.Errorf("manifest缺少必需配置声明，id=%s: %w", id, ErrManifestInvalid)
		}
	}
	return nil
}

func validateCompiledPack(pack *CompiledPack) error {
	if pack.Game.ID != pack.Manifest.ID {
		return fmt.Errorf("configs/game.json.id=%s与manifest.id=%s不一致: %w", pack.Game.ID, pack.Manifest.ID, ErrReferenceBroken)
	}

	resourceIDs, err := collectIDs("configs/resources.json.items", len(pack.Resources), func(index int) string { return pack.Resources[index].ID })
	if err != nil {
		return err
	}
	factionIDs, err := collectIDs("configs/factions.json.items", len(pack.Factions), func(index int) string { return pack.Factions[index].ID })
	if err != nil {
		return err
	}
	unitIDs, err := collectIDs("configs/units.json.items", len(pack.Units), func(index int) string { return pack.Units[index].ID })
	if err != nil {
		return err
	}
	if _, err := collectIDs("configs/buildings.json.items", len(pack.Buildings), func(index int) string { return pack.Buildings[index].ID }); err != nil {
		return err
	}
	terrainIDs, err := collectIDs("configs/terrains.json.items", len(pack.Terrains), func(index int) string { return pack.Terrains[index].ID })
	if err != nil {
		return err
	}
	mapIDs, err := collectIDs("configs/maps.json.items", len(pack.Maps), func(index int) string { return pack.Maps[index].ID })
	if err != nil {
		return err
	}

	if err := requireReference("configs/game.json.default_faction_id", pack.Game.DefaultFactionID, factionIDs); err != nil {
		return err
	}
	if err := requireReference("configs/game.json.default_map_id", pack.Game.DefaultMapID, mapIDs); err != nil {
		return err
	}
	for index, faction := range pack.Factions {
		if strings.TrimSpace(faction.Name) == "" {
			return fmt.Errorf("configs/factions.json.items[%d].name不能为空: %w", index, ErrFileInvalid)
		}
		if err := validateAmounts(fmt.Sprintf("configs/factions.json.items[%d].starting_resources", index), faction.StartingResources, resourceIDs, true); err != nil {
			return err
		}
	}
	for index, resource := range pack.Resources {
		if strings.TrimSpace(resource.Name) == "" {
			return fmt.Errorf("configs/resources.json.items[%d].name不能为空: %w", index, ErrFileInvalid)
		}
	}
	for index, unit := range pack.Units {
		if strings.TrimSpace(unit.Name) == "" {
			return fmt.Errorf("configs/units.json.items[%d].name不能为空: %w", index, ErrFileInvalid)
		}
		if err := requireReference(fmt.Sprintf("configs/units.json.items[%d].faction_id", index), unit.FactionID, factionIDs); err != nil {
			return err
		}
		if unit.Attack <= 0 || unit.Defense <= 0 || unit.Health <= 0 || unit.TrainTimeMs <= 0 {
			return fmt.Errorf("configs/units.json.items[%d]战斗属性和训练时长必须大于0: %w", index, ErrFileInvalid)
		}
		if err := validateAmounts(fmt.Sprintf("configs/units.json.items[%d].train_cost", index), unit.TrainCost, resourceIDs, false); err != nil {
			return err
		}
	}
	for index, building := range pack.Buildings {
		if strings.TrimSpace(building.Name) == "" {
			return fmt.Errorf("configs/buildings.json.items[%d].name不能为空: %w", index, ErrFileInvalid)
		}
		if err := requireReference(fmt.Sprintf("configs/buildings.json.items[%d].faction_id", index), building.FactionID, factionIDs); err != nil {
			return err
		}
		if building.BuildTimeMs <= 0 {
			return fmt.Errorf("configs/buildings.json.items[%d].build_time_ms必须大于0: %w", index, ErrFileInvalid)
		}
		if err := validateAmounts(fmt.Sprintf("configs/buildings.json.items[%d].build_cost", index), building.BuildCost, resourceIDs, false); err != nil {
			return err
		}
		if len(building.TrainableUnitIDs) == 0 {
			return fmt.Errorf("configs/buildings.json.items[%d].trainable_unit_ids不能为空: %w", index, ErrFileInvalid)
		}
		for unitIndex, unitID := range building.TrainableUnitIDs {
			if err := requireReference(fmt.Sprintf("configs/buildings.json.items[%d].trainable_unit_ids[%d]", index, unitIndex), unitID, unitIDs); err != nil {
				return err
			}
		}
	}
	for index, terrain := range pack.Terrains {
		if strings.TrimSpace(terrain.Name) == "" {
			return fmt.Errorf("configs/terrains.json.items[%d].name不能为空: %w", index, ErrFileInvalid)
		}
		if terrain.MoveCost <= 0 {
			return fmt.Errorf("configs/terrains.json.items[%d].move_cost必须大于0: %w", index, ErrFileInvalid)
		}
	}
	for index, gameMap := range pack.Maps {
		if strings.TrimSpace(gameMap.Name) == "" {
			return fmt.Errorf("configs/maps.json.items[%d].name不能为空: %w", index, ErrFileInvalid)
		}
		if gameMap.Width <= 0 || gameMap.Height <= 0 {
			return fmt.Errorf("configs/maps.json.items[%d]地图尺寸必须大于0: %w", index, ErrFileInvalid)
		}
		if err := requireReference(fmt.Sprintf("configs/maps.json.items[%d].default_terrain_id", index), gameMap.DefaultTerrainID, terrainIDs); err != nil {
			return err
		}
	}
	return nil
}

func collectIDs(path string, count int, getID func(index int) string) (map[string]struct{}, error) {
	if count == 0 {
		return nil, fmt.Errorf("%s至少需要一个条目: %w", path, ErrFileInvalid)
	}
	ids := make(map[string]struct{}, count)
	for index := 0; index < count; index++ {
		id := getID(index)
		if err := validateStableID(fmt.Sprintf("%s[%d].id", path, index), id); err != nil {
			return nil, err
		}
		if _, ok := ids[id]; ok {
			return nil, fmt.Errorf("%s[%d].id=%s重复: %w", path, index, id, ErrIDInvalid)
		}
		ids[id] = struct{}{}
	}
	return ids, nil
}

func validateStableID(path string, id string) error {
	if !stableIDPattern.MatchString(id) {
		return fmt.Errorf("%s=%s不符合小写短横线稳定ID格式: %w", path, id, ErrIDInvalid)
	}
	return nil
}

func requireReference(path string, id string, targets map[string]struct{}) error {
	if _, ok := targets[id]; !ok {
		return fmt.Errorf("%s=%s引用不存在: %w", path, id, ErrReferenceBroken)
	}
	return nil
}

func validateAmounts(path string, amounts map[string]int64, resourceIDs map[string]struct{}, allowZero bool) error {
	if len(amounts) == 0 {
		return fmt.Errorf("%s不能为空: %w", path, ErrFileInvalid)
	}
	for resourceID, amount := range amounts {
		if err := requireReference(path+"["+resourceID+"]", resourceID, resourceIDs); err != nil {
			return err
		}
		if amount < 0 || (!allowZero && amount == 0) {
			requirement := "正整数"
			if allowZero {
				requirement = "非负整数"
			}
			return fmt.Errorf("%s[%s]数量必须为%s: %w", path, resourceID, requirement, ErrFileInvalid)
		}
	}
	return nil
}

type version [3]int

func parseVersion(value string) (version, error) {
	parts := strings.Split(value, ".")
	if len(parts) != len(version{}) {
		return version{}, fmt.Errorf("版本%s必须使用主.次.修订格式", value)
	}
	var result version
	for index, part := range parts {
		number, err := strconv.Atoi(part)
		if err != nil || number < 0 || (len(part) > 1 && strings.HasPrefix(part, "0")) {
			return version{}, fmt.Errorf("版本%s的第%d段非法", value, index+1)
		}
		result[index] = number
	}
	return result, nil
}

func validateEngineConstraint(constraint EngineConstraint, engineVersion string) error {
	current, err := parseVersion(engineVersion)
	if err != nil {
		return fmt.Errorf("当前引擎版本非法: %w: %w", err, ErrEngineMismatch)
	}
	minimum, err := parseVersion(constraint.MinVersion)
	if err != nil {
		return fmt.Errorf("manifest.engine.min_version非法: %w: %w", err, ErrManifestInvalid)
	}
	maximum, err := parseVersion(constraint.MaxExclusive)
	if err != nil {
		return fmt.Errorf("manifest.engine.max_exclusive非法: %w: %w", err, ErrManifestInvalid)
	}
	if compareVersion(minimum, maximum) >= 0 {
		return fmt.Errorf("manifest.engine版本区间为空: %w", ErrManifestInvalid)
	}
	if compareVersion(current, minimum) < 0 || compareVersion(current, maximum) >= 0 {
		return fmt.Errorf("当前引擎版本%s不在[%s,%s)范围内: %w", engineVersion, constraint.MinVersion, constraint.MaxExclusive, ErrEngineMismatch)
	}
	return nil
}

func compareVersion(left version, right version) int {
	for index := range left {
		if left[index] < right[index] {
			return -1
		}
		if left[index] > right[index] {
			return 1
		}
	}
	return 0
}
