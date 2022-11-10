# Command Line Interface for `scribe`

## Preparation and design

### Decisions

* single executable file
* cross-platform
* parallel installation of R packages
* based on standard renv.lock file
* if no parameters add with command line then it will use `.scribe` configuration file or system variables
* logging capabilities

### Inspiration

* [Terraform](https://github.com/hashicorp/terraform)
* [Charm](https://github.com/charmbracelet)

### Std commands

#### List of all command line options

```bash
$ scribe  --help
```

#### Version of `scribe`

```bash
$ scribe  --version
```

#### List of all system variables used for `scribe`

```bash
$ scribe  --env
```

#### Installing packages from renv.lock file

```bash
$ scribe  renv.lock


Progress:
download  [###########----] 90/120
build     [########-------]  8/20
install   [###########----] 40/100
```

#### Installing packages from renv.lock URL

```bash
$ scribe  https://raw.githubusercontent.com/org/repo/blob/main/renv.lock


Progress:
download  [###########----] 90/120
build     [########-------]  8/20
install   [###########----] 40/100
```

If command will be run in `interactive mode` then progress bars will be displayed.

### Validating renv lock file

```bash
$scribe  --validate renv.lock

Error:
Not all packages in renv.lock file have Version parameter (Package: ggplot2).
```

### Changing the log level for CLI


Error (Only errors are displayed)
```bash
$ scribe  --log=error renv.lock
```

Warning (default log level, will should R warnings )

```bash
$ scribe  --log=warning renv.lock
```

Info (it shows information about completed steps)

```bash
$ scribe  --log=info renv.lock
```

Debug (Info level + messages from R command)

```bash
$ scribe  --log=debug renv.lock
```

Trace (Debug level + http requests)

```bash
$ scribe  --log=trace renv.lock
```

Quiet (will return 1 if there will be an error)

```bash
$ scribe  --quiet renv.lock
```

Short form

```bash
$ scribe  -q renv.lock
```

### Generating installation report

Installing packages from renv.lock file and generate report

```bash
$ scribe --report=html renv.lock

Progress:
download  [###########----] 90/120
build     [########-------]  8/20
install   [###########----] 40/100
```

`html` report is the default report type. We can shorten it to:

```bash
$ scribe --report renv.lock

Progress:
download  [###########----] 90/120
build     [########-------]  8/20
install   [###########----] 40/100
```

### Choosing which packages should go through `check` step

```bash
$ scribe --report --check-package teal,tern,teal.*  -check-as-cran  renv.lock

Progress:
download  [###########----] 90/120
build     [########-------]  8/20
install   [###########----] 40/100
check     [###------------]  2/8
```

`*` is a wildcard selector.

### Using configuration file

```bash
$ scribe renv.lock
```

Also, package could be mentioned in `.scribe` file:

```yaml
check:
  package:
  - teal
  - tern
  - teal.*
```

### Checking packages from repositories

```bash
$ scribe --report --check-remoteusername insightsengineering renv.lock

Progress:
download  [###########----] 90/120
build     [########-------]  8/20
install   [###########----] 40/100
check     [###------------]  2/8
```

Also, package could be mentioned in `.scribe` file:

```yaml
check:
  remoteusername:
  - insightsengineering
```

### Checking packags from RSPM Repository

```bash
$ scribe --report -check-repository NEST_RSPM  renv.lock
```

`.scribe` file:

```yaml
check:
  repository:
  - NEST_RSPM
```

### Checking packags base on multiple filters

```bash
$ scribe --report -check-filter repository=NEST_RSPM,remoteusername=insightsengineering  renv.lock
```

