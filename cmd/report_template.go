/*
Copyright 2023 F. Hoffmann-La Roche AG

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

const HTMLReportTemplate = `<!doctype html>
<html lang="en">

<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Scribe report</title>
    <link rel="stylesheet" href="https://cdn.datatables.net/1.13.4/css/dataTables.bootstrap5.min.css">
    <link rel="stylesheet" href="https://cdn.datatables.net/responsive/2.4.1/css/responsive.dataTables.min.css">
    <link rel="stylesheet" href="https://cdn.datatables.net/select/1.6.2/css/select.dataTables.min.css">
    <link rel="stylesheet" href="https://cdn.datatables.net/colreorder/1.6.2/css/colReorder.bootstrap5.min.css">
    <link rel="stylesheet" href="https://cdn.datatables.net/rowreorder/1.3.3/css/rowReorder.bootstrap5.min.css">
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/bootstrap@5.2.3/dist/css/bootstrap.min.css" integrity="sha384-rbsA2VBKQhggwzxH7pPCaAqO46MgnOM80zW1RWuH61DGLwZJEdK2Kadq2F9CUG65" crossorigin="anonymous">
    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.7.0/styles/base16/tomorrow.min.css" integrity="sha512-5D/fcZ3y3nuaeHSxDbFwWDEy1Fvj5qQKsU0tilD7bhWAA+IN/Jl9fzGdUotzvA7wgXtsnZmafcuunH+6nyuA0A==" crossorigin="anonymous" referrerpolicy="no-referrer"/>
    <!-- link rel="stylesheet" href="https://cdn.datatables.net/searchpanes/2.1.2/css/searchPanes.bootstrap5.min.css" -->
    <!-- Custom styles below -->
    <style>
    </style>
</head>

<body>
    <script src="https://code.jquery.com/jquery-3.6.4.min.js" integrity="sha256-oP6HI9z1XaZNBrJURtCoUT5SUnxFr8s3BzRl+cbzUq8=" crossorigin="anonymous"></script>
    <script src="https://cdn.datatables.net/1.13.4/js/jquery.dataTables.min.js"></script>
    <script src="https://cdn.datatables.net/responsive/2.4.1/js/dataTables.responsive.min.js"></script>
    <script src="https://cdn.datatables.net/select/1.6.2/js/dataTables.select.min.js"></script>
    <script src="https://cdn.datatables.net/1.13.4/js/dataTables.bootstrap5.min.js"></script>
    <script src="https://cdn.datatables.net/colreorder/1.6.2/js/dataTables.colReorder.min.js"></script>
    <script src="https://cdn.datatables.net/rowreorder/1.3.3/js/dataTables.rowReorder.min.js"></script>
    <script src="https://cdn.jsdelivr.net/npm/bootstrap@5.2.3/dist/js/bootstrap.min.js" integrity="sha384-cuYeSxntonz0PPNlHhBs68uyIAVpIIOZZ5JqeqvYYIcEL727kskC66kF92t6Xl2V" crossorigin="anonymous"></script>
    <script src="https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.7.0/highlight.min.js" integrity="sha512-bgHRAiTjGrzHzLyKOnpFvaEpGzJet3z4tZnXGjpsCcqOnAH6VGUx9frc5bcIhKTVLEiCO6vEhNAgx5jtLUYrfA==" crossorigin="anonymous" referrerpolicy="no-referrer"></script>
    <!-- script src="https://cdn.datatables.net/searchpanes/2.1.2/js/searchPanes.bootstrap5.min.js"></script -->
    <!-- script src="https://cdn.datatables.net/searchpanes/2.1.2/js/dataTables.searchPanes.min.js"></script -->
    <script>
        $(document).ready(function () {
            var table = $('#packagesTable').DataTable({
                select: true,
                responsive: true,
                colReorder: true,
                rowReorder: false,
                searchPanes: false
            });
            // table.searchPanes.container().prependTo(table.table().container());
            // table.searchPanes.resizePanes();
        });
        $(document).ready($(function () {
            $('#systemInfo').hide();
            $('#renvInfo').hide();
            $('#renvInfoOld').hide();
            $('#statusPage').show();
            $('#navbarSystemInformation').click(function () {
                $('#systemInfo').show();
                $('#renvInfo').hide();
                $('#renvInfoOld').hide();
                $('#statusPage').hide();
            });
            $('#navbarReport').click(function () {
                $('#systemInfo').hide();
                $('#renvInfo').hide();
                $('#renvInfoOld').hide();
                $('#statusPage').show();
            });
            $('#navbarRenvInformation').click(function () {
                $('#systemInfo').hide();
                $('#renvInfo').show();
                $('#renvInfoOld').hide();
                $('#statusPage').hide();
            });
            $('#navbarRenvInformationOld').click(function () {
                $('#systemInfo').hide();
                $('#renvInfo').hide();
                $('#renvInfoOld').show();
                $('#statusPage').hide();
            });
        }));
    </script>
    <script>hljs.highlightAll();</script>
    <nav class="navbar navbar-expand-lg bg-dark navbar-dark sticky-top">
        <div class="container-fluid">
            <div class="collapse navbar-collapse" id="navbarNav">
                <ul class="navbar-nav">
                    <li class="nav-item">
                        <a class="nav-link" href="#" id="navbarReport">Report</a>
                    </li>
                    <li class="nav-item">
                        <a class="nav-link" href="#" id="navbarSystemInformation">System information</a>
                    </li>
                    <li class="nav-item">
                        <a class="nav-link" href="#" id="navbarRenvInformation">Renv information</a>
                    </li>
                    <li class="nav-item">
                        <a class="nav-link" href="#" id="navbarRenvInformationOld">Renv information (without updated packages)</a>
                    </li>
                </ul>
            </div>
        </div>
    </nav>
    <div id="renvInfo" class="mt-3">
        <div class="container">
            <div class="row">
                <div class="col">
                    <p class="fw-bold">renv.lock filename</p>
                </div>
                <div class="col">
                    <code>
                    {{.RenvInformation.RenvFilename}}
                    </code>
                </div>
            </div>
            <div class="row">
                <div class="col">
                    <p class="fw-bold">renv.lock Contents</p>
                </div>
                <div class="col">
                    <pre><code>
                    {{.RenvInformation.RenvContents | safe}}
                    </code></pre>
                </div>
            </div>
        </div>
    </div>
    <div id="renvInfoOld" class="mt-3">
        <div class="container">
            <div class="row">
                <div class="col">
                <p class="fw-bold">renv.lock filename (without updated packages)</p>
                </div>
                <div class="col">
                    <code>
                    {{.RenvInformationOld.RenvFilename}}
                    </code>
                </div>
            </div>
            <div class="row">
                <div class="col">
                    <p class="fw-bold">renv.lock contents (without updated packages)</p>
                </div>
                <div class="col">
                    <pre><code>
                    {{.RenvInformationOld.RenvContents | safe}}
                    </code></pre>
                </div>
            </div>
        </div>
    </div>
    <div id="statusPage" class="mt-3">
        <table id="packagesTable" class="table table-striped table-bordered table-hover dt-responsive nowrap" style="width:100%">
            <thead>
                <tr>
                    <th>Package name</th>
                    <th>Package version</th>
                    <th>Download status</th>
                    <th>Build status</th>
                    <th>Install status</th>
                    <th>Check status</th>
                    <th>Check time (total: {{.TotalCheckTime}})</th>
                    <th>Package SHA</th>
                </tr>
            </thead>
            <tbody>
                <!-- go template iterating through all package information -->
                {{range .PackagesInformation}}
                <tr>
                    <td>{{.PackageName}}</td>
                    <td>{{.PackageVersion}}</td>
                    <td>{{.DownloadStatusText | safe}}</td>
                    <td>{{.BuildStatusText | safe}}</td>
                    <td>{{.InstallStatusText | safe}}</td>
                    <td>{{.CheckStatusText | safe}}</td>
                    <td>{{.CheckTime}}</td>
                    <td><code>{{.GitPackageShaOrRef}}</code></td>
                </tr>
                {{end}}
                <!-- end go template iteration -->
                <tfoot>
                    <tr>
                        <th>Package name</th>
                        <th>Package version</th>
                        <th>Download status</th>
                        <th>Build status</th>
                        <th>Install status</th>
                        <th>Check status</th>
                        <th>Check time (total: {{.TotalCheckTime}})</th>
                        <th>Package SHA</th>
                    </tr>
                </tfoot>
            </tbody>
        </table>
    </div>
    <div id="systemInfo" class="mt-3">
        <div class="container">
            <div class="row">
                <div class="col">
                    <p class="fw-bold">Operating system</p>
                </div>
                <div class="col">
                    <code>
                    {{.SystemInformation.OperatingSystem}}
                    </code>
                </div>
            </div>
            <div class="row">
                <div class="col">
                    <p class="fw-bold">Architecture</p>
                </div>
                <div class="col">
                    <code>
                    {{.SystemInformation.Architecture}}
                    </code>
                </div>
            </div>
            <div class="row">
                <div class="col">
                    <p class="fw-bold">Kernel version</p>
                </div>
                <div class="col">
                    <code>
                    {{.SystemInformation.KernelVersion}}
                    </code>
                </div>
            </div>
            <div class="row">
                <div class="col">
                    <p class="fw-bold">Pretty name</p>
                </div>
                <div class="col">
                    <code>
                    {{.SystemInformation.PrettyName}}
                    </code>
                </div>
            </div>
            <div class="row">
                <div class="col">
                    <p class="fw-bold">System packages</p>
                </div>
                <div class="col">
                    <code>
                        {{.SystemInformation.SystemPackages | safe}}
                    </code>
                </div>
            </div>
            <div class="row">
                <div class="col">
                    <p class="fw-bold">R version</p>
                </div>
                <div class="col">
                    <code>
                    {{.SystemInformation.RVersion}}
                    </code>
                </div>
            </div>
            <div class="row">
                <div class="col">
                    <p class="fw-bold">Time</p>
                </div>
                <div class="col">
                    <code>
                    {{.SystemInformation.Time}}
                    </code>
                </div>
            </div>
            <div class="row">
                <div class="col">
                    <p class="fw-bold">Environment variables</p>
                </div>
                <div class="col">
                    <code>
                    {{.SystemInformation.EnvVariables | safe}}
                    </code>
                </div>
            </div>
            <div class="row">
                <div class="col">
                    <p class="fw-bold">Hostname</p>
                </div>
                <div class="col">
                    <code>
                    {{.SystemInformation.Hostname}}
                    </code>
                </div>
            </div>
        </div>
    </div>

</body>

</html>
`
