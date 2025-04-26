package config

type EnvConfig struct {
	Name      string    `yaml:"name"`
	Value     string    `yaml:"value"`
	ValueFrom ValueFrom `yaml:"valueFrom"`
}

type ValueFrom struct {
	GCPSecretKeyRef GCPSecretKeyRef `yaml:"gcpSecretKeyRef"`
}

type GCPSecretKeyRef struct {
	Project string `yaml:"project"`
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
}

type DefaultsConfig struct {
	GCP GCPDefaults `yaml:"gcp"`
}

type GCPDefaults struct {
	Project string `yaml:"project"`
}

type RootConfig struct {
	Defaults DefaultsConfig `yaml:"defaults"`
	Env      []EnvConfig    `yaml:"env"`
}
