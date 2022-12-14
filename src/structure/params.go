package structure

type GlobalParams struct {
	LwopsPath string
}

type ApplyParams struct {
	PackagePath     string
	LoadWithAppPath string
	LoadAppVersion  string
}

type WebParams struct {
	Action     string
	WithRemove bool
}

type BuildParams struct {
	WebPort       int
	WebApiPort    int
	WebApiGateway string
}

type RollbackParams struct {
	Type string
}

type AppParams struct {
	ShowVersionList bool
	ToVersion       string
}

type EtcParams struct {
	ShowVersionList bool
	ToVersion       string
	WithBuild bool // 切换后是否生成环境参数
}

type ExecParams struct {
	Command string
	IsBackstage bool
}

type ServicedParams struct {
	Port   int
	Daemon bool
}
