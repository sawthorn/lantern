package config

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigSave(t *testing.T) {
	dir := "./test-data"
	port := 1234
	test_conf := Config{Dir: dir, Port: port}
	err := SaveConfig(test_conf)

	require.Nil(t, err)

	path, err := configPath()
	require.Nil(t, err)
	defer os.Remove(path)

	data, err := os.ReadFile(path)
	require.Nil(t, err)
	var conf Config
	json.Unmarshal(data, &conf)

	require.Equal(t, test_conf, conf)
}

func TestConfigLoad(t *testing.T) {
	dir := "./test-data"
	port := 1234
	test_conf := Config{Dir: dir, Port: port}

	path, err := configPath()
	require.Nil(t, err)
	defer os.Remove(path)
	data, err := json.Marshal(test_conf)
	require.Nil(t, err)
	err = os.WriteFile(path, data, 0777)
	require.Nil(t, err)

	conf, err := LoadConfig()
	require.Nil(t, err)
	require.Equal(t, test_conf, *conf)
}
