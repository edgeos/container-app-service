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

## Application Package Structure

The Container App Service supports both encrypted and unencrypted app package payloads.  Previous app package versions will not be compatable with this.

### Unencrypted Package Structure

- <application_name>.tar.gz: *Top-level package tarball*
  - MANIFEST.JSON: *LTC package metadata file*
  - <application_data>.tar.gz: *Tarball containing unencrypted application data*
    - docker-compose.yml: *Docker compose file for app*
    - <image1_name>.tar.gz: *Docker-save of application image*
    - <image2_name>.tar.gz: *Docker-save of application image*
    - <image..._name>.tar.gz: *Docker-save of application image*
    - <imageN_name>.tar.gz: *Docker-save of application image*
    - \<other folders and data\>: *Other directories and data can be included for the app and volume-mounted via compose file*

### Encrypted Package Structure

- <application_name>.tar.gz: *Top-level package tarball*
  - MANIFEST.JSON: *LTC package metadata file*
  - <machine1_name>.lockkey: *RSA encrypted symmetric key information for machine 1*
  - <machine2_name>.lockkey: *RSA encrypted symmetric key information for machine 2*
  - <machine..._name>.lockkey: *RSA encrypted symmetric key information for machine ...*
  - <machineN_name>.lockkey: *RSA encrypted symmetric key information for machine N*
  - <application_data>.tar.gz.enc: *Tarball containing encrypted application data*
    - docker-compose.yml: *Docker compose file for app*
    - <image1_name>.tar.gz: *Docker-save of application image*
    - <image2_name>.tar.gz: *Docker-save of application image*
    - <image..._name>.tar.gz: *Docker-save of application image*
    - <imageN_name>.tar.gz: *Docker-save of application image*
    - \<other folders and data\>: *Other directories and data can be included for the app and volume-mounted via compose file*

## Package Encryption

These instructions outline how one might generate an encrypted package for deployment.  For a quick reference see [Quick start encrypted package](#quick-start-encrypted-package). These steps can be included in a script or Makefile to automatically generate encrypted packages for an application, but this activity is left to the user, as applications vary in how they are built.

1. Prepare your payload file to encrypt as a tarball.  This should have the contents of <application_data>.tar.gz.enc as described in the Encrypted Package Structure subsection.  These instructions will use ```<application_data>.tar.gz``` as the name of the clear file and ```<application_data>.tar.gz.enc``` as the name of the encrypted version of this file.

1. Generate random symmetric key and initialization vector (iv).  This is a one-time use AES key and must be discarded after packaging is complete.

```
openssl rand -rand /dev/urandom 32 > aes.key
openssl rand -rand /dev/urandom 16 > aes.iv
```

1. Use the one-time use AES key and iv to encrypt the payload.

```
openssl enc -aes-256-cbc -in <application_data>.tar.gz  -out <application_data>.tar.gz.enc -K <key hex string> -iv <iv hex string>
```

Note: The <key hex string> and <iv hex string> in the command above MUST BE HEXADECIMAL STRING FORMAT.  These hex strings can be displayed from file using the ```xxd``` command like so:

```
xxd -p -c 9999 aes.<key/iv>
```

1. Concatenate padding, key, and iv into a single file with salt (16 random bits) at the start.  A the ```tools/write_padding.py``` python script can be used to generate the salt and padding as follows:

```
python3 write_padding.py <application_data>.tar.gz <application_data>.tar.gz.enc <machine_name>.clearkey
cat aes.key >> <machine1_name>.clearkey
cat aes.iv >> <machine1_name>.clearkey
```

Note: This is the clear text one-time use key for a single specific machine.  The process to generate this file must be repeated for each machine that this package may be deployed on so that each <machineX_name>.clearkey file has unique salt.

1. Encrypted the <machineX_name>.clearkey file(s) using the corresponding RSA public key for machineX.  This will generate a lockkey file called <machineX_name>

```
openssl rsautl -encrypt -inkey <machine1_name>.pubkey -pubin -in <machine1_name>.clearkey -out <machine1_name>
```

Note: Generation and configuration of these keys for the machine is discussed in the RSA Key Creation and Machine Commissioning.

1. Generate the top-level tarball file <application_name>.tar.gz, including the LTC manifest, all machine lockkey files, and the encrypted payload:

```
tar czf <application_name>.tar.gz MANIFEST.JSON <machine1_name> <machine2_name> <machineN_name> <application_data>.tar.gz.enc
```

The resulting <application_name>.tar.gz can now be deployed to the cappsd service.

## RSA Key Creation and Machine Commissioning

The package encryption strategy used by cappsd employs a one-time use AES key to encrypt sensitive application data.  This key is then encrypted using an asymmetric RSA public key that is paired with a private key stored on the target machine.  This key pair must be machine-specific and not re-used across machines.  This means that each machine needs to be comissioned with a key, and the corresponding public keys should be tracked by the packager.  The public/private RSA key pair can be generated with thhe following commands:

```
openssl genrsa -out <machine1_name>.privatekey 4096
openssl rsa -in <machine1_name>.privatekey -outform PEM -pubout -out <machine1_name>.pubkey
```

In the current version, TPM is not used to store the private key on the machine.  This is not secure.  TPM key retrieval will be added to a future version.  To commission the key on a machine in this version, update the ecs.json configuration file used by cappsd with a ```key_location``` field that has a path to the private key.  Store the private key at that location on the machine.

Example ecs.json:

```
{
    "listen_address": "/var/run/cappsd/cappsd.sock",
    "data_volume": "/mnt/data",
    "read_timeout": 30,
    "write_timeout:": 30,
    "key_location": "/key/<machine1_name>.privatekey",

    "Docker":
    {
        "endpoint": "unix://var/run/docker.sock",
        "reservedPort": 2375,
        "reservedSSLPort": 2376
    }
}
```

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

## Quick start encrypted package

Below is a sample script that can be used to create an encrypted package for container-app-service deployment.  This example assumes the user has downloaded a X.pubkey file which can be done on Local Management Console via the `Actions->Download Public` Key button, and placed the file at the root of this cloned repo.

```bash
#!/bin/bash

set -e

CAPPSD_IMG="${CAPPSD_IMG:-cappsd-img.tar.gz}"
APP_DATA="${APP_DATA:-application_data.tar.gz}"
PUBKEY="$(ls *.pubkey)"
MN=${PUBKEY/.*/}
MACHINE_NAME="${MACHINE_NAME:-${MN}}"

openssl rand -rand /dev/urandom 32 > aes.key
openssl rand -rand /dev/urandom 16 > aes.iv

openssl enc -aes-256-cbc -in ${APP_DATA} -out ${APP_DATA}.enc -K "$(xxd -p -c 9999 aes.key)" -iv "$(xxd -p -c 9999 aes.iv)"

python3 tools/write_padding.py ${APP_DATA} ${APP_DATA}.enc ${MACHINE_NAME}.clearkey
cat aes.key >> ${MACHINE_NAME}.clearkey
cat aes.iv >> ${MACHINE_NAME}.clearkey

openssl rsautl -encrypt -inkey ${MACHINE_NAME}.pubkey -pubin -in ${MACHINE_NAME}.clearkey -out ${MACHINE_NAME}

tar czf ${CAPPSD_IMG} MANIFEST.JSON ${MACHINE_NAME} ${APP_DATA}.enc
```


## Corrections and errors

Should you find any inconsistencies or errors in this document, kindly do one of the following:
1. Fork the repo, create your fixes, create a pull request with an explanation.
2. Create an issue on the repo from the ```Issues``` tab above the repo file navigator
3. Email <a href="mailto:edge.appdevdevops@ge.com">edge.appdevdevops@ge.com</a>
