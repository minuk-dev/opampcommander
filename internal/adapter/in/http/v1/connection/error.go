package connection

type UsecaseNotProvidedError struct {
	Usecase string
}

func (e *UsecaseNotProvidedError) Error() string {
	return "usecase not provided: " + e.Usecase
}
