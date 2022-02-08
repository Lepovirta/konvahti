# Konvahti

Konvahti is a watchdog software that fetches configuration files from remote sources periodically, and executes commands when the relevant files change.
This project exists to simplify configuration management for runtime environments ranging from bare metal to containers.
Here are some of its features.

**Pull based configuration synchronization!**
With Konvahti, you don't need to grant remote access to your computer to synchronize configurations to your systems.
This prevents the "god-mode" issue where a single system (e.g. CI/CD or configuration master) would have wide access to your systems.
It also allows you to block any network access to your systems.

**BYO configurer!**
Konvahti assumes nothing about what kind of files you pull from remote sources and what commands you run.
Instead, you can make Konvahti run any commands you like and read the remote files any way you like.

**Git and S3 support!**
Konvahti can pull configuration files from [Git](https://git-scm.com/) and S3 compatible (e.g. [S3](https://aws.amazon.com/s3/), [Minio](https://min.io/)) data sources.
Using the Git support, you can build your own GitOps with the configuration tools you like.

## Installation

Right now, the only way to install Konvahti is to build it from source.
First, you need to install [Go](https://go.dev/).
After that, you can install Konvahti using the following command.

```shell
go get gitlab.com/lepovirta/konvahti
```

Binary releases will come soon.

## License

[Apache 2.0](LICENSE)
