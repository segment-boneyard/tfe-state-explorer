package main

import prompt "github.com/c-bata/go-prompt"

func main() {
	t := NewTFELookup()

	p := prompt.New(
		t.Executor,
		t.Completer,
		prompt.OptionTitle("TFE lookup"),
		prompt.OptionPrefix(">>> "),
		prompt.OptionInputTextColor(prompt.Yellow),
	)

	p.Run()
}
