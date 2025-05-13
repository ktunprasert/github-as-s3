package application

type Application struct {
	token string
}

func NewApplication() *Application {
	return &Application{}
}

func NewApplicationWithOpts(opts ...applicationOpts) *Application {
	app := &Application{}
	for _, opt := range opts {
		opt(app)
	}

	return app
}

func (*Application) Start() error {
	// should be a goroutine

	return nil
}
