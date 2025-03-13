package plugins

type Plugin interface {
	Name() string
	Execute() error
	SetArgs(args []string)
}
