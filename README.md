# Konvahti

Konvahti is a watchdog software that fetches files from remote sources periodically, and executes commands when the relevant files change.
This project exists to simplify configuration management for runtime environments ranging from bare metal to containers.
Here are some of its features.

**Pull based configuration synchronization!**
With Konvahti, you don't need to grant remote access to your computer to synchronize configurations to your systems.
This prevents the "god-mode" issue where a single system (e.g. CI/CD or configuration master) would have wide access to your systems.
It also allows you to block any network access to your systems.

**BYO configurer!**
Konvahti assumes nothing about what kind of files you pull from remote sources and what commands you run.
Instead, you can make Konvahti run any commands you like and read the remote files any way you like.
Konvahti only depends on the executables you configure it to use, and executables don't have to be pulled dynamically from remote sources.

**Git and S3 support!**
Konvahti can pull configuration files from [Git](https://git-scm.com/) and S3 compatible (e.g. [S3](https://aws.amazon.com/s3/), [Minio](https://min.io/)) data sources.
Using the Git support, you can build your own GitOps with the configuration tools you like.

**Masterless!**
Konvahti doesn't have a remote software to control it.
Instead, it acts autonomously using the configurations it's launched with.
You will need a separate Git repository or an S3 bucket to pull files from though.

**No root access required!**
Konvahti can be run using any user that has permissions to write files to the directory you specify and run the commands you specify.

## How it works

For each watcher configuration, Konvahti follows this simple algorithm:

1. Pull latest files from a remote source to a local directory.
2. Check which actions should be run based on the file changes.
3. Run the commands of each matching action.
4. Repeat the cycle after specified time (interval) has elapsed.

When more than one watcher configuration is specified, the above algorithm is ran for each configuration concurrently.

## Installation

Right now, the only way to install Konvahti is to build it from source.
First, you need to install [Go](https://go.dev/).
After that, you can install Konvahti using the following command.

```shell
go get gitlab.com/lepovirta/konvahti
```

Binary releases will be coming soon.

## Usage

Konvahti accepts the following command-line arguments:

* `-config` flag followed by a path to a configuration file for Konvahti.
  When set to `-` or `STDIN`, the configuration is read from the STDIN stream.
  See the configuration section for further details.

Examples:

```shell
# Run konvahti with a configuration file
konvahti -config config.yaml

# Run konvahti with configurations loaded from STDIN
cat config.yaml | konvahti -config -
```

## Configuration

Konvahti is configured using [YAML](https://www.redhat.com/en/topics/automation/what-is-yaml) formatted files.
In the root of the configuration, you can specify the following settings.

**`watcher` (required):**
* List of watcher configurations
* At least one configuration is required
* All of the watchers are launched concurrently
* See the "Watcher" section below for more information

**`log` (optional):**
* Logging configuration to help tune the log output
* See the "Logging" section below for more information

### Watcher

A single watcher configuration specifies the remote source for files and the commands to run when those files change.
The watcher configurations are specified in the YAML field `watchers`.
The following settings are available.

**`interval` (optional):**
* How long to wait between each cycle.
* Accepts a string value in [Go duration format](https://pkg.go.dev/time#ParseDuration).
* When not set, the cycle is only ran once.
* Environment variable: `KONVAHTI_WATCHERS_N_INTERVAL` where `N` is the position of the watcher in the configuration.

**`refreshTimeout` (optional):**

* How long to allow Konvahti to wait for fetching the latest files from the remote source.
* Accepts a string value in [Go duration format](https://pkg.go.dev/time#ParseDuration).
* No timeout is used when this is not set.
* Environment variable: `KONVAHTI_WATCHERS_N_REFRESHTIMEOUT` where `N` is the position of the watcher in the configuration.

**`name` (optional):**

* Name of the configuration used for logging purposes.
* You can use any string value you like.
* By default, the index of the configuration is used here.
* Environment variable: `KONVAHTI_WATCHERS_N_NAME` where `N` is the position of the watcher in the configuration.

**`git` (optional):**

* Settings for a Git remote source
* Either this or the `s3` config must be specified
* See the "Git" section below for more information

**`s3` (optional):**

* Settings for a S3 remote source
* Either this or the `git` config must be specified
* See the "S3" section below for more information

**`actions` (optional):**

* List of actions to run when the remote source contents are fetched and changes are found
* At least one action must be specified
* See the "Actions" section below for more information

### Git

You can use a Git repository as a remote source for files to fetch on each cycle.
The Git configuration is specified in the YAML field `git`.
The following settings are available.

**`url` (required):**

* The URL for the remote Git repository
* Environment variable: `KONVAHTI_WATCHERS_N_GIT_URL` where `N` is the position of the watcher in the configuration.

**`branch` (required):**

* The name of the branch to track from the Git repository
* For example: `main`
* Environment variable: `KONVAHTI_WATCHERS_N_GIT_BRANCH` where `N` is the position of the watcher in the configuration.

**`directory` (required):**

* The local directory where the Git repository is to be cloned to
* Environment variable: `KONVAHTI_WATCHERS_N_GIT_DIRECTORY` where `N` is the position of the watcher in the configuration.


**`httpAuth` (optional):**

* HTTP authentication for Git. Includes the following fields.
* `username`: Username for the HTTP basic authentication
* `password`: Password for the HTTP basic authentication
* `token`: Token for HTTP token authentication. Supports OAuth bearer tokens.
* Use the username and password for GitHub, BitBucket, and GitLab instead of the token.
* Environment variables (`N` is the position of the watcher in the configuration)
  * `KONVAHTI_WATCHERS_N_GIT_HTTPAUTH_USERNAME`
  * `KONVAHTI_WATCHERS_N_GIT_HTTPAUTH_PASSWORD`
  * `KONVAHTI_WATCHERS_N_GIT_HTTPAUTH_TOKEN`

**`sshAuth` (optional):**

* SSH authentication for Git. Includes the following fields.
* `username`: Username for SSH authentication
* `keyPath`: Path to a SSH key on the file system to use for SSH authentication
* `keyPassword`: Password for the SSH key
* Environment variables (`N` is the position of the watcher in the configuration)
  * `KONVAHTI_WATCHERS_N_GIT_SSHAUTH_USERNAME`
  * `KONVAHTI_WATCHERS_N_GIT_SSHAUTH_KEYPATH`
  * `KONVAHTI_WATCHERS_N_GIT_SSHAUTH_KEYPASSWORD`

### S3

You can use a S3 bucket as a remote source for files to fetch on each cycle.
The names of the S3 objects are used as the local file paths.
The S3 configuration is specified in the YAML field `s3`.
The following settings are available.

**`endpoint` (required):**

* Endpoint URL for S3.
* If you plan on using AWS S3, see the [list of endpoints they provide](https://docs.aws.amazon.com/general/latest/gr/s3.html).
* If you plan on using a S3 compatible service, see the service provider's documentation for more details.
* Environment variable: `KONVAHTI_WATCHERS_N_S3_ENDPOINT` where `N` is the position of the watcher in the configuration.

**`accessKeyId` (required):**

* The ID part of the access key used for accessing S3
* Environment variable: `KONVAHTI_WATCHERS_N_S3_ACCESSKEYID` where `N` is the position of the watcher in the configuration.

**`secretAccessKey` (required):**

* The secret part of the access key used for accessing S3
* Environment variable: `KONVAHTI_WATCHERS_N_S3_SECRETACCESSKEY` where `N` is the position of the watcher in the configuration.

**`bucketName` (required):**

* Name of the S3 bucket to pull files from
* Environment variable: `KONVAHTI_WATCHERS_N_S3_BUCKETNAME` where `N` is the position of the watcher in the configuration.

**`directory` (required):**

* The local directory to use for storing all of the fetched S3 files
* Note that the latest S3 files will be found from the sub-directory `latest`
* Environment variable: `KONVAHTI_WATCHERS_N_S3_DIRECTORY` where `N` is the position of the watcher in the configuration.

**`bucketPrefix` (optional):**

* A prefix filter to use when fetching files from S3
* You can use this to limit fetching only certain "directory" of files from S3
* The prefix is automatically substracted from the local file paths
* By default, all files from the bucket are fetched
* Environment variable: `KONVAHTI_WATCHERS_N_S3_BUCKETPREFIX` where `N` is the position of the watcher in the configuration.

**`disableTls` (optional):**

* When set to `true`, TLS certificate checking is disabled
* This is intended for testing purposes only
* Default value: `false`
* Environment variable: `KONVAHTI_WATCHERS_N_S3_DISABLETLS` where `N` is the position of the watcher in the configuration.

### Actions

After fetching the latest files from the remote source, the list of changed files are compared to the actions specified in the configuration.
When there's a match, the action's commands are run.
The list of actions can be specified in the YAML field `actions`.
At least one action must be specified.
The following settings are available.

**`matchFiles` (optional):**

* List of glob patterns to match for fetched file changes.
* If file changes match any of the patterns, the action is executed.
* When empty (or unset), the action is executed every time any file changes.

**`command` (required):**

* Command and its arguments to run every time the action is triggered.
* Accepts a list of strings where the first entry is the command and the rest are arguments to that command.

**`preCommand` (optional):**

* Command and its arguments to run before the `command`.
* Intended for running preparations before the main command.
* Uses the same format as `command`.

**`postCommand` (optional):**

* Command and its arguments to run after the `command`.
* Intended for cleanup and reporting purposes after the main command.
* Run regardless if `command` (or `preCommand`) ran successfully or not.
* The post-command can read the status of the main command from the environment variable `KONVAHTI_ACTION_STATUS`, which is set to `success` on success, and `failure` on failure.
* Uses the same format as `command`.

**`env` (optional):**

* Environment variables to use with the action's commands provided in YAML key-value format.
* Default value: no environment variables set.

**`inheritAllEnvVars` (optional):**

* When set to `true` all of Konvahti's environment variables are passed to the action's commands.
* Default value: `false`

**`inheritEnvVars` (optional):**

* List of environment variable names from Konvahti's own environment variables to pass to the action's commands.
* Default value: no environment variables selected.

**`workDirectory` (optional):**

* The path to the directory where to run the action's commands in.
* The path is relative to the local directory specified in the remote source configuration.
* By default, the commands are run in the local directory specified in the remote source configuration.

**`timeout` (optional):**

* How long to allow Konvahti to wait for each action command to complete.
* Accepts a string value in [Go duration format](https://pkg.go.dev/time#ParseDuration).
* No timeout is used when this is not set.

**`maxRetries` (optional):**

* How many times to retry each action command.
* A command is retried when it exits with non-zero code.
* By default, no commands are retried.

**`name` (optional):**

* Name of the action used for logging purposes.
* You can use any string value you like.
* By default, the index of the action in the list is used here.

### Logging

Logging can be tuned using the `log` field in the root of the configuration file.
The following settings are available.

**`level` (optional):**

* The lowest priority level logs to include in the log output
* Available options: `trace`, `debug`, `info`, `warn`, `error`, `fatal`, `panic`, `disabled`
* Default value: `info`
* Environment variable: `KONVAHTI_LOG_LEVEL`

**`enablePrettyLogging` (optional):**

* When set to `true`, use text log output instead of JSON
* Available options: `true`, `false`
* Default value: `false`
* Environment variable: `KONVAHTI_LOG_ENABLEPRETTYLOGGING`

**`outputStream` (optional):**

* Stream to write logs to
* Available options: `stdout`, `stderr`
* Default value: `stderr`
* Environment variable: `KONVAHTI_LOG_OUTPUTSTREAM`

**`timestampFormat` (optional):**

* Format of the log timestamps
* Available options: `UNIXMS` (Unix timestamp in milliseconds), `UNIXMICRO` (Unix timestamp in microseconds), or any [Go time format](https://golangbyexample.com/time-date-formatting-in-go/)
* Default value: RFC3339 format
* Environment variable: `KONVAHTI_LOG_TIMESTAMPFORMAT`

**`timestampFieldName` (optional):**

* Name of the timestamp field in JSON log output
* Available options: any string
* Default value: `time`
* Environment variable: `KONVAHTI_LOG_TIMESTAMPFIELDNAME`

### Example

Using both Git remote and S3 remote.

```yaml
log:
  level: debug
  enablePrettyLogging: true
  outputStream: stdout
  timestampFormat: UNIXMS
  timestampFieldName: timestamp

watchers:
  - name: git_stuff
    interval: 1m
    refreshTimeout: 2m

    git:
      url: https://github.com/exampleorg/example.git
      branch: main
      directory: /var/lib/konvahti/gitstuff
      httpAuth:
        username: myusername
        password: supersecretpassword

    actions:

      - name: documentation
        matchFiles:
          - docs/*.md
        inheritEnvVars:
          - PATH
        workDirectory: docs
        command:
          - updatedocs
        postCommand:
          - report_to_slack.sh
          - docs
        maxRetries: 2
        timeout: 1m

      - name: deploy myapp
        matchFiles:
          - apps/myapp/config.yaml
        inheritAllEnvVars: true
        workDirectory: apps/myapp/
        preCommand:
          - prepare_app_env.py
        command:
          - deploy_app.py
        postCommand:
          - report_to_slack.sh
          - myapp
        maxRetries: 5
        timeout: 5m

  - name: s3_stuff
    interval: 1m
    refreshTimeout: 2m

    s3:
      endpoint: s3.eu-central-1.amazonaws.com
      accessKeyId: MYSUPERCOOLACCESSKEYID
      secretAccessKey: MYSUPERSECRETACCESSKEY
      bucketName: mysupercoolconfigbucket
      bucketPrefix: /mystuff/
      directory: /var/lib/konvahti/s3stuff

    actions:
      - name: deploy myapp
        matchFiles:
          - apps/myapp/config.yaml
        inheritAllEnvVars: true
        workDirectory: apps/myapp/
        preCommand:
          - prepare_app_env.py
        command:
          - deploy_app.py
        postCommand:
          - report_to_slack.sh
          - myapp
        maxRetries: 5
        timeout: 5m
```

## License

Copyright 2022 Jaakko Pallari

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

See [LICENSE](LICENSE) for further details.
