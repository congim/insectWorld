// Package gamepack 通用游戏包模块，负责加载、校验并编译题材内容配置。
package gamepack

// ManifestAPIVersionV1 首版manifest契约版本。
const ManifestAPIVersionV1 = 1

// 配置文档ID和schema常量，避免加载器与游戏包散落字符串。
const (
	ConfigIDGame      = "game"      // 游戏入口配置ID
	ConfigIDFactions  = "factions"  // 阵营集合配置ID
	ConfigIDResources = "resources" // 资源集合配置ID
	ConfigIDUnits     = "units"     // 单位集合配置ID
	ConfigIDBuildings = "buildings" // 建筑集合配置ID
	ConfigIDTerrains  = "terrains"  // 地形集合配置ID
	ConfigIDMaps      = "maps"      // 地图集合配置ID

	SchemaGameV1      = "game/v1"      // 游戏入口schema
	SchemaFactionsV1  = "factions/v1"  // 阵营集合schema
	SchemaResourcesV1 = "resources/v1" // 资源集合schema
	SchemaUnitsV1     = "units/v1"     // 单位集合schema
	SchemaBuildingsV1 = "buildings/v1" // 建筑集合schema
	SchemaTerrainsV1  = "terrains/v1"  // 地形集合schema
	SchemaMapsV1      = "maps/v1"      // 地图集合schema
)

// Manifest 游戏包清单，声明版本、模块和配置文件，不承载玩法数据。
type Manifest struct {
	APIVersion int              `yaml:"api_version"` // manifest契约版本，当前仅支持1
	ID         string           `yaml:"id"`          // 游戏包稳定ID，使用小写短横线格式
	Name       string           `yaml:"name"`        // 游戏包展示名称，仅用于管理界面
	Version    string           `yaml:"version"`     // 游戏包语义版本，格式为主.次.修订
	Engine     EngineConstraint `yaml:"engine"`      // 兼容的引擎版本区间
	Modules    []string         `yaml:"modules"`     // 启用模块列表，只允许已知模块ID
	Configs    []ConfigFile     `yaml:"configs"`     // 配置文档声明列表
}

// EngineConstraint 游戏包兼容的引擎版本区间，采用左闭右开语义。
type EngineConstraint struct {
	MinVersion   string `yaml:"min_version"`   // 最低兼容版本，包含该版本
	MaxExclusive string `yaml:"max_exclusive"` // 最高兼容版本，不包含该版本
}

// ConfigFile manifest中的配置文件声明。
type ConfigFile struct {
	ID       string `yaml:"id"`       // 配置文档ID，对应ConfigIDXxx常量
	Path     string `yaml:"path"`     // 相对游戏包根目录的JSON文件路径
	Schema   string `yaml:"schema"`   // 配置schema标识，对应SchemaXxxV1常量
	Required bool   `yaml:"required"` // 是否为当前包启动所必需
}

// GameConfig 游戏入口配置，定义默认阵营和地图引用。
type GameConfig struct {
	ID               string `json:"id"`                 // 游戏内容稳定ID，应与manifest ID一致
	DefaultFactionID string `json:"default_faction_id"` // 新玩家默认阵营ID，引用factions
	DefaultMapID     string `json:"default_map_id"`     // 新世界默认地图ID，引用maps
}

// FactionList 阵营配置文档。
type FactionList struct {
	Items []FactionConfig `json:"items"` // 阵营定义列表，ID在文档内唯一
}

// FactionConfig 阵营配置，题材名称和初始资源均属于游戏包内容。
type FactionConfig struct {
	ID                string           `json:"id"`                 // 阵营稳定ID
	Name              string           `json:"name"`               // 阵营展示名称
	StartingResources map[string]int64 `json:"starting_resources"` // 初始资源，key引用resources，value为整数数量
}

// ResourceList 资源配置文档。
type ResourceList struct {
	Items []ResourceConfig `json:"items"` // 资源定义列表，ID在文档内唯一
}

// ResourceConfig 资源定义。
type ResourceConfig struct {
	ID   string `json:"id"`   // 资源稳定ID
	Name string `json:"name"` // 资源展示名称
}

// UnitList 单位配置文档。
type UnitList struct {
	Items []UnitConfig `json:"items"` // 单位定义列表，ID在文档内唯一
}

// UnitConfig 单位定义，首版只包含垂直切片需要的训练与战斗基础值。
type UnitConfig struct {
	ID          string           `json:"id"`            // 单位稳定ID
	Name        string           `json:"name"`          // 单位展示名称
	FactionID   string           `json:"faction_id"`    // 所属阵营ID，引用factions
	Attack      int64            `json:"attack"`        // 基础攻击力，必须大于0
	Defense     int64            `json:"defense"`       // 基础防御力，必须大于0
	Health      int64            `json:"health"`        // 单位生命值，必须大于0
	TrainCost   map[string]int64 `json:"train_cost"`    // 单位训练消耗，key引用resources
	TrainTimeMs int64            `json:"train_time_ms"` // 单位训练时长，单位毫秒且必须大于0
}

// BuildingList 建筑配置文档。
type BuildingList struct {
	Items []BuildingConfig `json:"items"` // 建筑定义列表，ID在文档内唯一
}

// BuildingConfig 建筑定义，首版覆盖建造成本和可训练单位引用。
type BuildingConfig struct {
	ID               string           `json:"id"`                 // 建筑稳定ID
	Name             string           `json:"name"`               // 建筑展示名称
	FactionID        string           `json:"faction_id"`         // 所属阵营ID，引用factions
	BuildCost        map[string]int64 `json:"build_cost"`         // 建造消耗，key引用resources
	BuildTimeMs      int64            `json:"build_time_ms"`      // 建造时长，单位毫秒且必须大于0
	TrainableUnitIDs []string         `json:"trainable_unit_ids"` // 可训练单位ID列表，引用units
}

// TerrainList 地形配置文档。
type TerrainList struct {
	Items []TerrainConfig `json:"items"` // 地形定义列表，ID在文档内唯一
}

// TerrainConfig 地形定义。
type TerrainConfig struct {
	ID       string `json:"id"`        // 地形稳定ID
	Name     string `json:"name"`      // 地形展示名称
	MoveCost int64  `json:"move_cost"` // 进入地形的移动力消耗，必须大于0
	Blocked  bool   `json:"blocked"`   // 是否阻挡普通移动
}

// MapList 地图配置文档。
type MapList struct {
	Items []MapConfig `json:"items"` // 地图定义列表，ID在文档内唯一
}

// MapConfig 地图定义，首版使用整数格子尺寸和默认地形。
type MapConfig struct {
	ID               string `json:"id"`                 // 地图稳定ID
	Name             string `json:"name"`               // 地图展示名称
	Width            int32  `json:"width"`              // 地图宽度，单位格且必须大于0
	Height           int32  `json:"height"`             // 地图高度，单位格且必须大于0
	DefaultTerrainID string `json:"default_terrain_id"` // 默认地形ID，引用terrains
}

// CompiledPack 校验通过的强类型游戏包，供装配层只读使用。
type CompiledPack struct {
	Root      string           // 游戏包根目录的绝对路径
	Manifest  Manifest         // 已校验的manifest
	Game      GameConfig       // 游戏入口配置
	Factions  []FactionConfig  // 阵营配置列表
	Resources []ResourceConfig // 资源配置列表
	Units     []UnitConfig     // 单位配置列表
	Buildings []BuildingConfig // 建筑配置列表
	Terrains  []TerrainConfig  // 地形配置列表
	Maps      []MapConfig      // 地图配置列表
}
