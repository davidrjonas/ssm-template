ssm-template
============

A simple template renderer with support for [AWS Systems Manager Parameter Store](https://docs.aws.amazon.com/systems-manager/latest/userguide/systems-manager-paramstore.html). Based on the venerable [confd](https://github.com/kelseyhightower/confd), dramatically simplified, adding [Masterminds sprig functions](https://github.com/Masterminds/sprig)_.

Resources
---------

- [confd template functions](https://github.com/kelseyhightower/confd/blob/master/docs/templates.md)
- [sprig template functions](http://masterminds.github.io/sprig/))

Usage
-----

There are no options or configuration. AWS credentials are set in the usual way via `~/.aws/credentials` and environment variables.

Read from STDIN, write to STDOUT, errors to STDERR, exit code set.

```bash
$ cat <<EOF > my.conf.tpl
Sprig now: {{ now }}
confd getenv: {{ getenv "HOME" }}

{{ with get "/config/host" -}}
confd get: {{ .Key }}={{ .Value }}
{{- end }}

confd gets:
{{- range gets "/config" }}
- key: {{ .Key }}
  value: {{ .Value }}
{{- end }}
EOF

$ AWS__REGION=us-west-1 ssm-template < my.conf.tpl | tee my.conf
Sprig now: 2019-02-22 14:32:51.121105 -0800 PST m=+0.039965546
confd getenv: /Users/djonas

confd get: /config/host=127.0.0.1

confd gets:
- key: /config/host
  value: 127.0.0.1
- key: /config/port
  value: 80
```
