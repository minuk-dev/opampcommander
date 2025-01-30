package connection

type Option func(*Controller)

func WithConnectionUsecase(connectionUsecase Usecase) Option {
	return func(c *Controller) {
		c.connectionUsecase = connectionUsecase
	}
}
