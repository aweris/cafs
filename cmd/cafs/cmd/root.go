package cmd

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "cafs",
	Short: "Content-Addressable File System CLI",
	Long:  "CLI for managing CAFS namespaces and syncing with OCI registries.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().String("config", "", "config file (default: ~/.config/cafs/config.yaml)")
	rootCmd.PersistentFlags().String("cache-dir", "", "cache directory (default: ~/.local/share/cafs)")

	viper.BindPFlag("cache_dir", rootCmd.PersistentFlags().Lookup("cache-dir"))
}

func initConfig() {
	if cfg := rootCmd.PersistentFlags().Lookup("config").Value.String(); cfg != "" {
		viper.SetConfigFile(cfg)
	} else {
		viper.AddConfigPath(configDir())
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	viper.SetEnvPrefix("CAFS")
	viper.AutomaticEnv()
	viper.SetDefault("cache_dir", defaultCacheDir())

	viper.ReadInConfig()
}

func configDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "cafs")
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".config", "cafs")
	}
	return ".cafs"
}

func defaultCacheDir() string {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "cafs")
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".local", "share", "cafs")
	}
	return ".cafs"
}

func getCacheDir() string {
	return viper.GetString("cache_dir")
}
