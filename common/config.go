package common

// Config is the structure describing common configurtion
// for all parts of the solution (management-node,
// worker-node, schedctl)
type Config struct {
	LogLvl  string `yaml:"log_level"`
	LogPath string `yaml:"log_path"`
	Stdout  bool   `yaml:"stdout"`

	TmpFolderPath string `yaml:"tmp_path"`
}
