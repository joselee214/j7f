package config

import (
	"github.com/spf13/viper"
)

//type Config interface {
//	// Set sets the value for the key in the override regiser.
//	// Set is case-insensitive for a key.
//	// Will be used instead of values obtained via
//	// flags, config file, ENV, default, or key/value store.
//	Set(key string, value interface{})
//
//	// AddConfigPath adds a path for Viper to search for the config file in.
//	// Can be called multiple times to define multiple search paths.
//	AddConfigPath(in string)
//
//	// AllKeys returns all keys holding a value, regardless of where they are set.
//	// Nested keys are returned with a v.keyDelim (= ".") separator
//	AllKeys() []string
//
//	// AllSettings merges all settings and returns them as a map[string]interface{}.
//	AllSettings() map[string]interface{}
//
//	// Get can retrieve any value given the key to use.
//	// Get is case-insensitive for a key.
//	// Get has the behavior of returning the value associated with the first
//	// place from where it is set. Viper will check in the following order:
//	// override, flag, env, config file, key/value store, default
//	//
//	// Get returns an interface. For a specific value use one of the Get____ methods.
//	Get(key string) interface{}
//
//	// GetBool returns the value associated with the key as a boolean.
//	GetBool(key string) bool
//
//	// GetDuration returns the value associated with the key as a duration.
//	GetDuration(key string) time.Duration
//
//	// GetFloat64 returns the value associated with the key as a float64.
//	GetFloat64(key string) float64
//
//	// GetInt32 returns the value associated with the key as an integer.
//	GetInt32(key string) int32
//
//	// GetInt64 returns the value associated with the key as an integer.
//	GetInt64(key string) int64
//
//	// GetString returns the value associated with the key as a string.
//	GetString(key string) string
//
//	// GetStringMap returns the value associated with the key as a map of interfaces.
//	GetStringMap(key string) map[string]interface{}
//
//	// GetStringMapString returns the value associated with the key as a map of strings.
//	GetStringMapString(key string) map[string]string
//
//	// GetStringMapStringSlice returns the value associated with the key as a map to a slice of strings.
//	GetStringMapStringSlice(key string) map[string][]string
//
//	// GetStringSlice returns the value associated with the key as a slice of strings.
//	GetStringSlice(key string) []string
//
//	// GetTime returns the value associated with the key as time.
//	GetTime(key string) time.Time
//
//	WatchConfig()
//
//	OnConfigChange(run func(in fsnotify.Event))
//
//	// ReadInConfig will discover and load the configuration file from disk
//	// and key/value stores, searching in one of the defined paths.
//	ReadInConfig() error
//
//	// ConfigFileUsed returns the file used to populate the config registry.
//	ConfigFileUsed() string
//
//	// WriteConfig writes the current configuration to a file.
//	WriteConfig() error
//
//	// WriteConfigAs writes current configuration to a given filename.
//	WriteConfigAs(filename string) error
//}

func NewViper() *Configer {
	v := viper.New()
	return &Configer{v}
}
