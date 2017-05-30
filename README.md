# Container App Service
This service is responsible for managing and deploying single/multi-container
applications.   

This has been tested on OS X and Linux.  Depends on Docker to build.


## Building

Run `make` or `make build` to compile your app.  This will use a Docker image
to build your app, with the current directory volume-mounted into place.  This
will store incremental state for the fastest possible build.  Run `make
all-build` to build for all architectures.

Run `make image` to build the image for local testing.

Run `make test` to test the image.  Format checks, lint, and unit tests will be
executed.

Run `make deploy` to push the binary to Artifactory.  

Run `make clean` to clean up.

This [section](https://github.build.ge.com/PredixEdgeProjects/template-c#jenkins-integration)
will cover how to set up a GitHub project for automated testing.

## TODO
- [ ] Migrate from godep to glide, gb or other package management scheme to streamline future development

- [ ] Finalize container monitoring of event stream.  This will support monitoring and restart of failed service along with any dependencies

- [ ] Make 1st class systemd service to integrate tighter with EdgeOS

- [ ] Reduce footprint by supporting "dockerless" implementation (i.e. containerd / runc)

- [ ] Move from Docker compose packaging to more generic pod specification to provide future flexibility

```json
{
  "kind": "Pod",
  "apiVersion": "v1",
  "metadata": {
    "name": "",
    "labels": {
      "name": ""
    },
    "generateName": "",
    "namespace": "",
    "annotations": []
  },
  "spec": {
      "containers": [
        {
          "name": "",
          "image": "",
          "command": [
            ""
          ],
          "args": [
            ""
          ],
          "env": [
            {
              "name": "",
              "value": ""
            }
          ],
          "imagePullPolicy": "",
          "workdingDir": "",
          "depends_on": [
            ""
          ],
          "external_links": [
              ""
          ],
          "networks": [
              ""
          ],
          "ports": [
            {
              "containerPort": 0,
              "name": "",
              "protocol": ""
            }
          ],
          "resources": {
              "cpu": "",
              "memory": ""
          }
          "livenessProbe":
          {
              "httpGet":
              {
                  "path": "",
                  "port": 0,
                  "httpHeaders": ""
              },
              "initialDelaySeconds": 0,
              "timeoutSeconds": 0
          }
      }
      ],
      "restartPolicy": "",
      "volumes": [
        {
          "name": "",
          "hostPath": ""
        }
        ],
    }
  }
}
```

## Corrections and errors

Should you find any inconsistencies or errors in this document, kindly do one of the following:
1. Fork the repo, create your fixes, create a pull request with an explanation.
2. Create an issue on the repo from the ```Issues``` tab above the repo file navigator
3. Email <a href="mailto:edge.appdevdevops@ge.com">edge.appdevdevops@ge.com</a>
