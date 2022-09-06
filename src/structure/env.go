package structure

type EnvironmentParams struct {
	Build struct {
		BuildTime     string `yaml:"buildTime"`
		WebPort       int    `yaml:"webPort"`
		WebApiPort    int    `yaml:"webApiPort"`
		WebApiGateWay string `yaml:"webApiGateWay"`
		MacAddr       string `yaml:"macAddr"`
	} `yaml:"build"`

	Uid int `yaml:"uid"` // 操作用户的uid
}
