package action

type Action struct {
	Code string
}

func New(code string) *Action {
	return &Action{
		Code: code,
	}
}
