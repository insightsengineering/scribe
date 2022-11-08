# Command Line Interface for `scribe`

## Preparation and design

### Decisions

* single executable file
* cross-platform
* parallel installation of R packages
* base on standard renv.lock file
* if no parameters add with command line then it will use `.scribe` configuration file or system variables
* logging capabilities

### Inspiration

* https://github.com/hashicorp/terraform
* https://github.com/charmbracelet


### Std commands

#### Installing packages from renv.lock file

```bash
$ scribe  -help
```

#### Version of `scribe`

```bash
$ scribe  -version
```

#### List of all system variables used for `scribe`

```bash
$ scribe  -env
```

#### Installing packages from renv.lock file

```bash
$ scribe  renv.lock


Progress:
download  [###########----] 90/120
build     [########-------]  8/20
install   [###########----] 40/100
```

### Validating renv lock file

```bash
$scribe  -validate renv.lock

Error:
Not all packages in renv.lock file has Version parameter (Package: ggplot2).
```

### Changing log lavel for cli

Info (info. about done steps)

```bash
$ scribe  -v renv.lock
```

Debug (Info level + messages from R command)

```bash
$ scribe  -vv renv.lock
```

Track (Debug level + http requests)

```bash
$ scribe  -vvv renv.lock
```

Quiet (will return 1 if there will be an error)

```bash
$ scribe  -q renv.lock
```

### Generating installation report

Installing packages from renv.lock file and generate report

```bash
$ scribe -report renv.lock

Progress:
download  [###########----] 90/120
build     [########-------]  8/20
install   [###########----] 40/100
```

### Choosing which packages should go through `check` step

```bash
$ scribe -report -check-Package teal,tern,teal.*  -check-as-cran  renv.lock

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
  Package:
  - teal
  - tern
  - teal.*
```

### Checking packages from repositories

```bash
$ scribe -report -check-RemoteUsername insightsengineering renv.lock

Progress:
download  [###########----] 90/120
build     [########-------]  8/20
install   [###########----] 40/100
check     [###------------]  2/8
```

Also, package could be mentioned in `.scribe` file:

```yaml
check:
  RemoteUsername:
  - insightsengineering
```

### Checking packags from RSPM Repository

```bash
$ scribe -report -check-Repository NEST_RSPM  renv.lock
```

`.scribe` file:

```yaml
check:
  Repository:
  - NEST_RSPM
```
