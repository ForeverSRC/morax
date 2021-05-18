package loadbalance

type Balance interface {
	DoBalance([]string) (string, error)
}
