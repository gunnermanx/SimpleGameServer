package strategy

type ELO struct {
}

func NewELOStrategy() (elo *ELO) {
	elo = &ELO{}
	return
}
