package pkg3

type Sys interface {
	Exists(path string) bool
	Exec(dir, cmd string, params ...string) ([]byte, error)
	Move(src, dst string) error
	Copy(src, dst string) error
	Preppend(file string, lines []string) error
}
