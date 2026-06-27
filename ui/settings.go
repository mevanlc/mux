package ui

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const configFileName = "config.json"

var userConfigDir = os.UserConfigDir

type splitLayout string

const (
	layoutHorizontal splitLayout = "horizontal"
	layoutVertical   splitLayout = "vertical"
)

type settings struct {
	SplitSizes map[splitLayout]float64 `json:"split_sizes"`
}

func defaultSettings() settings {
	return settings{
		SplitSizes: map[splitLayout]float64{
			layoutHorizontal: defaultSplitRatio,
			layoutVertical:   defaultSplitRatio,
		},
	}
}

func loadSettings() settings {
	path, err := settingsPath()
	if err != nil {
		return defaultSettings()
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return defaultSettings()
	}

	var result settings
	if err := json.Unmarshal(data, &result); err != nil {
		return defaultSettings()
	}

	return normalizedSettings(result)
}

func saveSettings(s settings) error {
	s = normalizedSettings(s)

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

func saveSplitRatio(layout splitLayout, ratio float64) error {
	s := loadSettings()
	s.setSplitRatio(layout, ratio)
	return saveSettings(s)
}

func settingsPath() (string, error) {
	dir, err := userConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "mux", configFileName), nil
}

func normalizedSettings(s settings) settings {
	result := defaultSettings()
	for layout, ratio := range s.SplitSizes {
		result.setSplitRatio(layout, ratio)
	}
	return result
}

func (s settings) splitRatio(layout splitLayout) float64 {
	if ratio, ok := s.SplitSizes[layout]; ok {
		return validSplitRatio(ratio)
	}
	return defaultSplitRatio
}

func (s *settings) setSplitRatio(layout splitLayout, ratio float64) {
	if s.SplitSizes == nil {
		s.SplitSizes = make(map[splitLayout]float64)
	}
	s.SplitSizes[layout] = validSplitRatio(ratio)
}

func validSplitRatio(ratio float64) float64 {
	if ratio <= 0 || ratio >= 1 {
		return defaultSplitRatio
	}
	return ratio
}
