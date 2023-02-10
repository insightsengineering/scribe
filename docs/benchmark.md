# Benchmark of scribe

## Explanation of terms

“Number of Threads” is the maximum number of simultaneously running
installations of R packages.

“Highly Dependent Package” is an R package which has a lot dependencies
on other R packages.

“Non-Dependent Package” is an R package without dependencies on other R
packages.

“Low Dependent Package” is an R package which has a few (1 or 2)
dependencies on Non-Dependent Packages.

### Disclaimer

“scribe” is a very complex program that performs many disk read-write
operations as well as queries to web services.

“scribe” invokes “R CMD\` check where execution times can vary greatly.

The results obtained in this test may not match your results, because
they depend on parameters of the server on which it was run, speed of
your internet connection and the choice of R packages.

## Installing packages with different numbers of threads

The more threads, the shorter the time for installation of all packages.
This relationship is not linear.

We get to the point where the extra threads don’t speed up the overall
process. It is because Highly Dependent Packages (with tightly coupled
packages) do not allow parallelisation. The last packages on the
installation list are run sequentially.

At the beginning of the installation process, Non-Dependent Packages are
being installed first. They do not have dependencies, so parallelisation
is done quite effectively.

<img src="benchmark_files/figure-markdown_github/unnamed-chunk-2-1.png" alt="Performance on average number of packages"  />

## Installing packages and Strongly Connected Packages

For a large number of packages, Strongly Connected Packages may appear.
Strongly Connected Packages are a group of packages, where if one wants
to use a single package, the imperative is to install a cohort of other
packages. Examples: “dplyr” with “tibble”, “rlang”, “vctrs” or
“devtools” with “cli”, “pkgdown”, “rcmdcheck”, “remotes”, “roxygen2”.

If there is a Highly Dependent Package in the group, then, in fact, we
are dealing with Strongly Connected Packages.

The topological sorting algorithm considers Strongly Connected Packages
to parallelize effectively. This is not always the case, as some
packages do not have a complete description of all the packages they
require for installation. In launches, we can observe it, when there is
an increased number of threads and the installation execution time
decreases significantly.

<img src="benchmark_files/figure-markdown_github/unnamed-chunk-3-1.png" alt="Performance on large number of packages"  />

## renv and scribe

In summary, “scribe” can install packages much faster than “renv” will
ever do. Even if, “renv” will use “pak” package for downloading
packages, “scribe” will do it faster, because it uses parallelization
for download, build, and installation process.

<img src="benchmark_files/figure-markdown_github/unnamed-chunk-4-1.png" alt="renv and scribe"  />

## Checks in scribe

We missed one feature in the renv. It was running “R CMD checks” on
several packages. We have parallelized this procedure.

This is the result for running only checks.

<img src="benchmark_files/figure-markdown_github/unnamed-chunk-5-1.png" alt="Checks in scribe"  />
