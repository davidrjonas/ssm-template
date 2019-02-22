package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"text/template"

	"github.com/Masterminds/sprig"

	confdssm "github.com/kelseyhightower/confd/backends/ssm"
)

type KVPair struct {
	Key   string
	Value string
}

func main() {
	bytes, err := ioutil.ReadAll(os.Stdin)

	if err != nil {
		fmt.Fprintln(os.Stderr, "reading standard input:", err)
		os.Exit(1)
	}

	client, err := NewClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to create aws client:", err)
		os.Exit(1)
	}

	tplfuncs := newFuncMap()
	tplfuncs["get"] = client.Get
	tplfuncs["gets"] = client.GetAll
	tplfuncs["getv"] = client.GetValue
	tplfuncs["getvs"] = client.GetAllValues

	err = template.Must(
		template.New("stdin").Funcs(sprig.TxtFuncMap()).Funcs(tplfuncs).Parse(string(bytes)),
	).Execute(os.Stdout, "none")

	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to render template: ", err)
		os.Exit(1)
	}
}

type Client struct {
	client *confdssm.Client
}

func NewClient() (*Client, error) {
	c, err := confdssm.New()
	if err != nil {
		return nil, err
	}

	return &Client{client: c}, nil
}

func (c *Client) Get(name string) (interface{}, error) {
	kv, err := c.client.GetValues([]string{name})
	if err != nil {
		return "", err
	}

	return KVPair{Key: name, Value: kv[name]}, nil
}

func (c *Client) GetValue(name string, v ...string) (interface{}, error) {
	kv, err := c.client.GetValues([]string{name})
	if err != nil {
		if len(v) > 0 {
			return v[0], nil
		}
		return "", err
	}

	return kv[name], nil
}

func (c *Client) GetAll(name string) (interface{}, error) {
	kv, err := c.client.GetValues([]string{name})
	if err != nil {
		return "", err
	}

	ks := make([]KVPair, len(kv))

	i := 0
	for k := range kv {
		ks[i] = KVPair{Key: k, Value: kv[k]}
		i++
	}

	return ks, nil
}

func (c *Client) GetAllValues(name string) (interface{}, error) {
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
