package config

type EnvConfig struct {
	Name      string    `yaml:"name"`
	Value     string    `yaml:"value"`
	ValueFrom ValueFrom `yaml:"valueFrom"`
}

type ValueFrom struct {
	GCPSecretKeyRef      GCPSecretKeyRef      `yaml:"gcpSecretKeyRef"`
	GitlabVariableKeyRef GitlabVariableKeyRef `yaml:"gitlabVariableKeyRef"`
	GithubVariableKeyRef GithubVariableKeyRef `yaml:"githubVariableKeyRef"`
}

type GithubVariableKeyRef struct {
	Repo string `yaml:"repo"`
	Name string `yaml:"name"`
}

type GCPSecretKeyRef struct {
	Project string `yaml:"project"`
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
}

type GitlabVariableKeyRef struct {
	Project string `yaml:"project"`
	Key     string `yaml:"key"`
}

type DefaultsConfig struct {
	GCP    GCPDefaults    `yaml:"gcp"`
	Github GithubDefaults `yaml:"github"`
}

type GCPDefaults struct {
	Project string `yaml:"project"`
}

type GithubDefaults struct {
	Repo string `yaml:"repo"`
}

type RootConfig struct {
	Defaults DefaultsConfig `yaml:"defaults"`
	Env      []EnvConfig    `yaml:"env"`
}
