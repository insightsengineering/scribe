# Benchmark of scribe

#### Disclaimer

“scribe” is very complex program that performs many disk read-write
operations as well as queries to web services.

“scribe” is invoking “R CMD\` where execution times can vary greatly.

The results obtained in this test may not match your results, because it
depends on parameters of the server on which it was run, speed of your
internet connection and the choices of R packages.

#### Explanation of terms

“Number of Threads” is the maximum number of simultaneously running
installations of R packages.

“Highly Dependent Package” is a R package which has a lot dependencies
to other R packages

“Non-dependent Package” is a R package which no dependencies on other R
packages

“Low Dependent Package” is a R package which has few (1 or 2 )
dependencies on Non-dependent Packages

## Installing packages with different numbers of threads

The more threads, the shorter the installation of all packages. This
relationship is not linear.

We get to the point where the extra threads don’t speed up the overall
process. It is so, because Highly Dependent Package which has tightly
coupled packages do not allow parallelisation. The last packages in the
installation list are run sequentially.

At the beginning of the installation process, Non-dependent Package are
being installed as a first packages. They do not have dependencies, so
parallelisation is done quite effectively.

![](benchmark_files/figure-markdown_github/unnamed-chunk-2-1.png)

## Installing packages and Strongly Connected Packages

For a large number of packages, Strongly Connected Packages may appear.
Strongly Connected Packages are a group of packages, where if he wants
to use one package, he will have to install a whole horde of other
packages E.g. would be “dplyr” with “tibble”, “rlang”, “vctrs” or
“devtools” with “cli”, “pkgdown”, “rcmdcheck”, “remotes”, “roxygen2”.

If there is a Highly Dependent Package then there should be Strongly
Connected Packages.

The topological sorting algorithm considers Strongly Connected Packages
to parallelize effectively. This is not always the case, as some
packages do not have a complete description of all the packages they
would need for installation. In launches, we can observe such an
occurrence, when there is an increased number of threads and the
installation execution time decreases significantly.

![](benchmark_files/figure-markdown_github/unnamed-chunk-3-1.png)

## renv and scribe

In summary, “scribe” can install packages much faster than “renv” will
ever do. Even if, “renv” will use “pak” package for downloading
packages, “scribe” will be faster, because it used parallelization for
download, build, installation process.

![](benchmark_files/figure-markdown_github/unnamed-chunk-4-1.png)

## Checks in “scribe”

We missed one feature in the renv. It was running “R cmd checks” on
several packages. We have parallelized this procedure.

Here are the result for running only checks.

![](benchmark_files/figure-markdown_github/unnamed-chunk-5-1.png)
