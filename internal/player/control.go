package player

type mpvControl interface {
	args() []string
	command(string) error
	request(string) (string, error)
	close()
}
