# AWX Client Examples

This directory contains a collections of examples that show how to use the AWX
client.

## Running the Examples

In order to run the examples you will need to provide the details to connect to
your AWX server, either modifiying the source code of the example or using the
command line options. For example, to run the example that lists the job
templates you can use the following command line:

```bash
$ go run list_job_templates.go \
-url "https://awx.example.com/api" \
-username "admin" \
-password "..." \
-ca-file "ca.pem" \
-logtostderr \
-v=2
```

Note that the `-logtostderr` and `-v=2` options aren't needed, but they are very
convenient for development, as they ensure that all the HTTP traffic is dumped
to the terminal.
