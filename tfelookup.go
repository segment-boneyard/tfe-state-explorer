package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"

	prompt "github.com/c-bata/go-prompt"
	"github.com/segmentio/tfe-state-explorer/tfstate"
)

const (
	TFAddr = "https://atlas.hashicorp.com"
)

type tfelookup struct {
	envs        []string
	state       *tfstate.State
	flat        map[string]tfstate.AttributeOrOutput
	getPrompts  []prompt.Suggest
	loadPrompts []prompt.Suggest
}

func NewTFELookup() *tfelookup {
	token, ok := os.LookupEnv("ATLAS_TOKEN")
	if !ok {
		log.Fatal("Must set $ATLAS_TOKEN")
	}

	// populate available envs
	envs, err := GetEnvs(token)
	if err != nil {
		log.Fatal(err)
	}

	t := &tfelookup{
		envs:       envs,
		state:      nil,
		flat:       nil,
		getPrompts: nil,
	}

	t.genLoadPrompts()
	return t
}

func (t *tfelookup) LoadEnv(env string) {
	url := fmt.Sprintf("%s/api/v1/terraform/state/%s",
		TFAddr, env)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Printf("failed to load env")
		return
	}

	req.Header.Add("X-Atlas-Token", os.Getenv("ATLAS_TOKEN"))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("failed to load env")
		return
	}
	defer resp.Body.Close()

	state, err := tfstate.ReadState(resp.Body)
	if err != nil {
		fmt.Printf("failed to load env: %s\n", err.Error())
		return
	}

	t.state = state
	t.flat = state.FlattenAttributesAndOutputs()
	t.genGetPrompts()
	fmt.Printf("loaded env %s\n", env)
}

func (t *tfelookup) genGetPrompts() {
	prompts := []prompt.Suggest{}

	keys := []string{}
	for k := range t.flat {
		keys = append(keys, k)
	}

	sort.Sort(ByLength(keys))

	for _, k := range keys {
		s := prompt.Suggest{Text: k}
		prompts = append(prompts, s)
	}
	t.getPrompts = prompts
}

func (t *tfelookup) genLoadPrompts() {
	prompts := []prompt.Suggest{}

	for _, e := range t.envs {
		s := prompt.Suggest{Text: e}
		prompts = append(prompts, s)
	}

	t.loadPrompts = prompts
}

func (t *tfelookup) Get(k string) {
	if t.state == nil || t.flat == nil {
		fmt.Println("must load environment first")
		return
	}

	val, ok := t.flat[k]
	if !ok {
		fmt.Println("value not found for path")
		return
	}

	fmt.Printf("%s\n", toString(val.Value))
	return
}

func toString(v interface{}) string {
	switch out := v.(type) {
	case string:
		return out
	case []interface{}:
		return formatList(out)
	// TODO: better map support
	default:
		return fmt.Sprintf("%+v", out)
	}
}

func formatList(vals []interface{}) string {
	parts := []string{}

	for _, v := range vals {
		parts = append(parts, toString(v))
	}

	return strings.Join(parts, ",")
}

func (t *tfelookup) Executor(s string) {
	s = strings.TrimSpace(s)

	parts := strings.Split(s, " ")
	if len(parts) == 0 {
		return
	}

	command := parts[0]

	switch command {
	case "":
		return
	case "quit", "exit":
		fmt.Println("Bye!")
		os.Exit(0)
		return
	case "load":
		if len(parts) < 2 {
			fmt.Println("must pass an environment to load")
			return
		}
		t.LoadEnv(parts[1])
		return
	case "get":
		if len(parts) < 2 {
			fmt.Println("must pass an argument to get")
			return
		}
		t.Get(parts[1])
		return
	default:
		fmt.Printf("unrecognized command '%s'", command)
	}
	return
}

func (t *tfelookup) Completer(d prompt.Document) []prompt.Suggest {
	if d.TextBeforeCursor() == "" {
		return []prompt.Suggest{}
	}

	args := strings.Split(d.TextBeforeCursor(), " ")
	w := d.GetWordBeforeCursor()

	if len(args) <= 1 {
		return prompt.FilterHasPrefix(commands, args[0], true)
	}

	command := args[0]

	switch command {
	case "get":
		if t.getPrompts != nil {
			return prompt.FilterHasPrefix(t.getPrompts, w, true)
		}
	case "load":
		if t.loadPrompts != nil {
			return prompt.FilterHasPrefix(t.loadPrompts, w, true)
		}
	}
	return []prompt.Suggest{}
}

var commands = []prompt.Suggest{
	{Text: "get", Description: "Get value for terraform path"},
	{Text: "load", Description: "Load a terraform environment"},
	{Text: "quit", Description: "Quit this program"},
}

type Environment struct {
	Username string `json:"username"`
	Name     string `json:"name"`
}

type AvailableState struct {
	Environment Environment `json:"environment"`
}

type AtlasStateResponse struct {
	States []AvailableState `json:"states"`
}

func GetEnvs(token string) ([]string, error) {
	url := fmt.Sprintf("%s/api/v1/terraform/state", TFAddr)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return []string{}, err
	}

	req.Header.Add("X-Atlas-Token", os.Getenv("ATLAS_TOKEN"))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return []string{}, err
	}
	defer resp.Body.Close()

	d := json.NewDecoder(resp.Body)
	var response AtlasStateResponse
	if err := d.Decode(&response); err != nil {
		return []string{}, err
	}

	envs := []string{}
	for _, s := range response.States {
		name := fmt.Sprintf("%s/%s", s.Environment.Username, s.Environment.Name)
		envs = append(envs, name)
	}

	return envs, nil
}

type ByLength []string

func (a ByLength) Len() int           { return len(a) }
func (a ByLength) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByLength) Less(i, j int) bool { return len(a[i]) < len(a[j]) }
