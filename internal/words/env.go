package words

type Environment interface {
	Resolve(string) ([]string, error)
	Define(string, []string) error
	Delete(string) error
}
