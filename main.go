package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"

	"text/template"

	"github.com/Masterminds/sprig"

	confdssm "github.com/kelseyhightower/confd/backends/ssm"
)

type kvPair struct {
	Key   string
	Value string
}

func main() {
	input, err := ioutil.ReadAll(os.Stdin)

	if err != nil {
		fmt.Fprintln(os.Stderr, "reading standard input:", err)
		os.Exit(1)
	}

	client, err := newClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to create aws client:", err)
		os.Exit(1)
	}

	tplfuncs := newFuncMap()
	tplfuncs["get"] = client.get
	tplfuncs["gets"] = client.getAll
	tplfuncs["getv"] = client.getValue
	tplfuncs["getvs"] = client.getAllValues

	output := bytes.NewBuffer([]byte{})
	err = template.Must(
		template.New("stdin").Funcs(sprig.TxtFuncMap()).Funcs(tplfuncs).Parse(string(input)),
	).Execute(output, "none")

	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to render template: ", err)
		os.Exit(1)
	}

	output.WriteTo(os.Stdout)
}

type client struct {
	client *confdssm.Client
}

func newClient() (*client, error) {
	c, err := confdssm.New()
	if err != nil {
		return nil, err
	}

	return &client{client: c}, nil
}

func (c *client) get(name string) (interface{}, error) {
	kv, err := c.client.GetValues([]string{name})
	if err != nil {
		return "", err
	}

	return kvPair{Key: name, Value: kv[name]}, nil
}

func (c *client) getValue(name string, v ...string) (interface{}, error) {
	kv, err := c.client.GetValues([]string{name})
	if err != nil {
		if len(v) > 0 {
			return v[0], nil
		}
		return "", err
	}

	return kv[name], nil
}

func (c *client) getAll(name string) (interface{}, error) {
	kv, err := c.client.GetValues([]string{name})
	if err != nil {
		return "", err
	}

	ks := make([]kvPair, len(kv))

	i := 0
	for k := range kv {
		ks[i] = kvPair{Key: k, Value: kv[k]}
		i++
	}

	return ks, nil
}

func (c *client) getAllValues(name string) (interface{}, error) {
	kv, err := c.client.GetValues([]string{name})
	if err != nil {
		return "", err
	}

	ks := make([]string, len(kv))

	i := 0
	for k := range kv {
		ks[i] = kv[k]
		i++
	}

	return ks, nil
}
