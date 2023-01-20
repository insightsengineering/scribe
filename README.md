# scribe

[![build](https://github.com/insightsengineering/scribe/actions/workflows/build.yml/badge.svg)](https://github.com/insightsengineering/scribe/actions/workflows/build.yml)

`scribe` (acronym for `s`ystem `c`ompatibility `r`eport for `i`nstall & `b`uild `e`valuation) is a project that creates complete build, check and install reports for a collection of R projects that are defined in an [renv.lock](https://rstudio.github.io/renv/articles/lockfile.html) file.

## Installing

Simply download the project for your distribution from the [releases](https://github.com/insightsengineering/scribe/releases) page. `scribe` is distributed as a single binary file and does not require any additional system requirements other than `git` and `R`, both of which it integrates and interfaces with externally.

## Usage

`scribe` is a command line utility, so after installing the binary in your `PATH`, simply run the following command to view its capabilities:

```bash
scribe --help
```

Example usage with multiple flags:
```bash
scribe --interactive --logLevel debug --maskedEnvVars 'password|key' --renvLockFilename renv2.lock --checkPackage 'tern*,teal*' --reportDir htmlreportdir --maxDownloadRoutines 100 --maxCheckRoutines 20 --numberOfWorkers 150
```

## Cache

`scribe` uses cache stored in `/tmp/scribe` for various purposes.

The results of download, installation, build and check stages are stored in `/tmp/scribe/cache`. When `scribe` detects presence of files with such results, it skips respective stages.

In order to run the download, installation, build and check from scratch, the `/tmp/scribe/cache` directory should be removed manually. Removing whole `/tmp/scribe` directory is also possible - in that case, the packages will have to be downloaded again because cached `tar.gz` packages and `git` repositories are stored in this directory.

## Development

This project is built with the [Go programming language](https://go.dev/).

### Development Environment

It is recommended to use Go v1.19+ for developing this project. This project uses a pre-commit configuration and it is recommended to [install and use pre-commit](https://pre-commit.com/#install) when you are developing this project.

### Common Commands

Run `make help` to list all related targets that will aid local development.

### Style

This project adopts the [Uber styleguide](https://github.com/uber-go/guide/blob/master/style.md).

## License

`scribe` is licensed under the Apache 2.0 license. See [LICENSE.md](LICENSE.md) for details.
