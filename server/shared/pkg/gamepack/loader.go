// Package gamepack 通用游戏包模块，负责加载、校验并编译题材内容配置。
package gamepack

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const manifestFileName = "manifest.yaml"

// LoadAndCompile 从目录加载游戏包，完成严格解码、契约校验和强类型编译。
func LoadAndCompile(root string, engineVersion string) (*CompiledPack, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("解析游戏包根目录失败，root=%s: %w", root, err)
	}

	manifest, err := loadManifest(filepath.Join(absRoot, manifestFileName))
	if err != nil {
		return nil, err
	}
	if err := validateManifest(absRoot, manifest, engineVersion); err != nil {
		return nil, err
	}

	pack := &CompiledPack{Root: absRoot, Manifest: manifest}
	for _, file := range manifest.Configs {
		if err := loadConfig(absRoot, file, pack); err != nil {
			return nil, err
		}
	}
	if err := validateCompiledPack(pack); err != nil {
		return nil, err
	}
	return pack, nil
}

func loadManifest(path string) (Manifest, error) {
	file, err := os.Open(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("读取manifest失败，文件=%s: %w: %w", path, err, ErrManifestInvalid)
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	decoder.KnownFields(true)
	var manifest Manifest
	if err := decoder.Decode(&manifest); err != nil {
		return Manifest{}, fmt.Errorf("解析manifest失败，文件=%s: %w: %w", path, err, ErrManifestInvalid)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return Manifest{}, fmt.Errorf("manifest只能包含一个YAML文档，文件=%s: %w", path, ErrManifestInvalid)
	}
	return manifest, nil
}

func loadConfig(root string, file ConfigFile, pack *CompiledPack) error {
	path, err := resolveConfigPath(root, file.Path)
	if err != nil {
		return err
	}

	switch file.ID {
	case ConfigIDGame:
		return decodeJSON(path, &pack.Game)
	case ConfigIDFactions:
		var value FactionList
		if err := decodeJSON(path, &value); err != nil {
			return err
		}
		pack.Factions = value.Items
	case ConfigIDResources:
		var value ResourceList
		if err := decodeJSON(path, &value); err != nil {
			return err
		}
		pack.Resources = value.Items
	case ConfigIDUnits:
		var value UnitList
		if err := decodeJSON(path, &value); err != nil {
			return err
		}
		pack.Units = value.Items
	case ConfigIDBuildings:
		var value BuildingList
		if err := decodeJSON(path, &value); err != nil {
			return err
		}
		pack.Buildings = value.Items
	case ConfigIDTerrains:
		var value TerrainList
		if err := decodeJSON(path, &value); err != nil {
			return err
		}
		pack.Terrains = value.Items
	case ConfigIDMaps:
		var value MapList
		if err := decodeJSON(path, &value); err != nil {
			return err
		}
		pack.Maps = value.Items
	default:
		return fmt.Errorf("不支持的配置文档，id=%s: %w", file.ID, ErrManifestInvalid)
	}
	return nil
}

func resolveConfigPath(root string, relativePath string) (string, error) {
	if relativePath == "" || filepath.IsAbs(relativePath) {
		return "", fmt.Errorf("配置路径必须是非空相对路径，path=%s: %w", relativePath, ErrFileInvalid)
	}
	cleanPath := filepath.Clean(relativePath)
	if cleanPath == "." || cleanPath == ".." || strings.HasPrefix(cleanPath, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("配置路径越出游戏包目录，path=%s: %w", relativePath, ErrFileInvalid)
	}
	path := filepath.Join(root, cleanPath)
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("配置路径越出游戏包目录，path=%s: %w", relativePath, ErrFileInvalid)
	}
	return path, nil
}

func decodeJSON(path string, target any) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("读取配置失败，文件=%s: %w: %w", path, err, ErrFileInvalid)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("解析配置失败，文件=%s: %w: %w", path, err, ErrFileInvalid)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return fmt.Errorf("配置文件只能包含一个JSON值，文件=%s: %w", path, ErrFileInvalid)
	}
	return nil
}
