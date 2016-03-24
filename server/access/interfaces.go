package access

type Access interface {
	Login(username, password string) (User, error)
}

type User interface {
	Name() string
	Groups() []string
	PrimaryGroup() string
}
