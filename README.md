# Auto-heal Service

This project contains the _auto-heal_ service. It receives alert
notitifications from the Prometheus alert manager and executes Ansible
playbooks to resolve the root cause.

## Building

To build the binary run this command:

```
$ make
```

To build the RPM and the images, run this command:

```
make build-images
```
