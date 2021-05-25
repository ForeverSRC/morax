package check

type CheckConfig struct {
	CheckPort       int    `mapstructre:"checkPort"`
	CheckUri        string `mapstructure:"checkUri"`
	Timeout         string `mapstructure:"timeout"`
	Interval        string `mapstructure:"interval"`
	DeregisterAfter string `mapstructure:"deregisterAfter"`
}
