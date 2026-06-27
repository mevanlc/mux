package ui

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const configFileName = "config.json"

var userConfigDir = os.UserConfigDir

type settings struct {
	HorizontalSplit float64 `json:"horizontal_split"`
	VerticalSplit   float64 `json:"vertical_split"`
}

func defaultSettings() settings {
	return settings{
		HorizontalSplit: defaultSplitRatio,
		VerticalSplit:   defaultSplitRatio,
	}
}

func loadSettings() settings {
	result := defaultSettings()

	path, err := settingsPath()
	if err != nil {
		return result
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return result
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return defaultSettings()
	}

	result.HorizontalSplit = validSplitRatio(result.HorizontalSplit)
	result.VerticalSplit = validSplitRatio(result.VerticalSplit)
	return result
}

func saveSettings(s settings) error {
	s.HorizontalSplit = validSplitRatio(s.HorizontalSplit)
	s.VerticalSplit = validSplitRatio(s.VerticalSplit)

	path, err := settingsPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0644)
}

func settingsPath() (string, error) {
	dir, err := userConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "mux", configFileName), nil
}

func validSplitRatio(ratio float64) float64 {
	if ratio <= 0 || ratio >= 1 {
		return defaultSplitRatio
	}
	return ratio
}
