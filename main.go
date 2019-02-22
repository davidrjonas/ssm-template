package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/cloudflare/cfssl/log"
)

type KVPair struct {
	Key   string
	Value string
}

type KVPairs []KVPair

func main() {
	bytes, err := ioutil.ReadAll(os.Stdin)

	if err != nil {
		fmt.Fprintln(os.Stderr, "reading standard input:", err)
		os.Exit(1)
	}

	client, err := NewAwsClient()
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

type AwsClient struct {
	ssm *ssm.SSM
}

func NewAwsClient() (*AwsClient, error) {
	sess := session.Must(session.NewSession())

	// Fail early, if no credentials can be found
	_, err := sess.Config.Credentials.Get()
	if err != nil {
		return nil, err
	}

	return &AwsClient{ssm: ssm.New(sess, nil)}, nil
}

// TODO add default fallback
func (c *AwsClient) Getv(name string, def ...string) (interface{}, error) {
	params := &ssm.GetParameterInput{
		Name:           aws.String(name),
		WithDecryption: aws.Bool(true),
	}

	resp, err := c.ssm.GetParameter(params)

	if err != nil {
		return nil, err
	}

	if *resp.Parameter.Value == "" {
		return def, nil
	}

	return *resp.Parameter.Value, nil
}

func (c *AwsClient) Get(name string) (interface{}, error) {
	kv, err := c.getParameter(name)
	if err != nil {
		return "", err
	}

	return KVPair{Key: name, Value: kv[name]}, nil
}

func (c *AwsClient) GetValue(name string, v ...string) (interface{}, error) {
	kv, err := c.getParameter(name)
	if err != nil {
		if len(v) > 0 {
			return v[0], nil
		}
		return "", err
	}

	return kv[name], nil
}

func (c *AwsClient) GetAll(name string) (interface{}, error) {
	v, err := c.getParametersWithPrefix(name)
	if err != nil {
		return "", err
	}

	ks := make([]KVPair, len(v))

	i := 0
	for k := range v {
		ks[i] = KVPair{Key: k, Value: v[k]}
		i++
	}

	return ks, nil
}

func (c *AwsClient) GetAllValues(name string) (interface{}, error) {
	v, err := c.getParametersWithPrefix(name)
	if err != nil {
		return "", err
	}

	ks := make([]string, len(v))

	i := 0
	for k := range v {
		ks[i] = v[k]
		i++
	}

	return ks, nil
}

func (c *AwsClient) GetValues(keys []string) (map[string]string, error) {
	vars := make(map[string]string)
	var err error
	for _, key := range keys {
		log.Debug("Processing key=%s", key)
		var resp map[string]string
		resp, err = c.getParametersWithPrefix(key)
		if err != nil {
			return vars, err
		}
		if len(resp) == 0 {
			resp, err = c.getParameter(key)
			if err != nil && err.(awserr.Error).Code() != ssm.ErrCodeParameterNotFound {
				return vars, err
			}
		}
		for k, v := range resp {
			vars[k] = v
		}
	}
	return vars, nil
}

func (c *AwsClient) getParametersWithPrefix(prefix string) (map[string]string, error) {
	var err error
	parameters := make(map[string]string)
	params := &ssm.GetParametersByPathInput{
		Path:           aws.String(prefix),
		Recursive:      aws.Bool(true),
		WithDecryption: aws.Bool(true),
	}
	c.ssm.GetParametersByPathPages(params,
		func(page *ssm.GetParametersByPathOutput, lastPage bool) bool {
			for _, p := range page.Parameters {
				parameters[*p.Name] = *p.Value
			}
			return !lastPage
		})
	return parameters, err
}

func (c *AwsClient) getParameter(name string) (map[string]string, error) {
	parameters := make(map[string]string)
	params := &ssm.GetParameterInput{
		Name:           aws.String(name),
		WithDecryption: aws.Bool(true),
	}
	resp, err := c.ssm.GetParameter(params)
	if err != nil {
		return parameters, err
	}
	parameters[*resp.Parameter.Name] = *resp.Parameter.Value
	return parameters, nil
}
