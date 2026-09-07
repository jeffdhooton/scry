package graphfixture

type Speaker interface {
	Speak() string
}

type Greeter struct{}

func (Greeter) Speak() string { return "hello" }

func Invoke(s Speaker) string { return s.Speak() }

func Unrelated() int { return 7 }
