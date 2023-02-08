/*
Copyright 2022 F. Hoffmann-La Roche AG

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
    <link rel="stylesheet" href="https://cdn.datatables.net/1.13.1/css/jquery.dataTables.min.css">
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/bootstrap@5.2.3/dist/css/bootstrap.min.css"
        integrity="sha384-rbsA2VBKQhggwzxH7pPCaAqO46MgnOM80zW1RWuH61DGLwZJEdK2Kadq2F9CUG65" crossorigin="anonymous">
</head>

<body>
    <script src="https://code.jquery.com/jquery-3.5.1.js"></script>
    <script src="https://cdn.datatables.net/1.13.1/js/jquery.dataTables.min.js"></script>
    <script src="https://cdn.jsdelivr.net/npm/bootstrap@5.1.3/dist/js/bootstrap.bundle.min.js"
        integrity="sha384-ka7Sk0Gln4gmtz2MlQnikT1wXgYsOg+OMhuP+IlRH9sENBO0LRn5q+8nbTov4+1p"
        crossorigin="anonymous"></script>
    <script>
        $(document).ready(function () {
            $('#packagesTable').DataTable();
        });
        $(document).ready($(function () {
            $('#systemInfo').hide();
            $('#statusPage').show();
            $('#renvInfo').hide();
            $('#navbarSystemInformation').click(function () {
                $('#statusPage').hide();
                $('#renvInfo').hide();
                $('#systemInfo').show();
            });
            $('#navbarReport').click(function () {
                $('#systemInfo').hide();
                $('#renvInfo').hide();
                $('#statusPage').show();
            });
            $('#navbarRenvInformation').click(function () {
                $('#systemInfo').hide();
                $('#renvInfo').show();
                $('#statusPage').hide();
            });
        }));
    </script>
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
                </ul>
            </div>
        </div>
    </nav>
    <div id="renvInfo">
        <div class="container">
            <div class="row">
                <div class="col">
                renv.lock filename
                </div>
                <div class="col">
                    <p class="font-monospace">
                    {{.RenvInformation.RenvFilename}}
                    </p>
                </div>
            </div>
            <div class="row">
                <div class="col">
                renv.lock contents
                </div>
                <div class="col">
                    <p class="font-monospace">
                    {{.RenvInformation.RenvContents | safe}}
                    </p>
                </div>
            </div>
        </div>
    </div>
    <div id="statusPage">
        <table id="packagesTable" class="display">
            <thead>
                <tr>
                    <th>Package name</th>
                    <th>Package version</th>
                    <th>Download status</th>
                    <th>Build status</th>
                    <th>Install status</th>
                    <th>Check status</th>
                    <th>Check time (total: {{.TotalCheckTime}})</th>
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
                </tr>
                {{end}}
            </tbody>
        </table>
    </div>
    <div id="systemInfo">
        <div class="container">
            <div class="row">
                <div class="col">
                    <p class="fw-bold">Operating system</p>
                </div>
                <div class="col">
                    {{.SystemInformation.OperatingSystem}}
                </div>
            </div>
            <div class="row">
                <div class="col">
                    <p class="fw-bold">Architecture</p>
                </div>
                <div class="col">
                    {{.SystemInformation.Architecture}}
                </div>
            </div>
            <div class="row">
                <div class="col">
                    <p class="fw-bold">Kernel version</p>
                </div>
                <div class="col">
                    {{.SystemInformation.KernelVersion}}
                </div>
            </div>
            <div class="row">
                <div class="col">
                    <p class="fw-bold">Pretty name</p>
                </div>
                <div class="col">
                    {{.SystemInformation.PrettyName}}
                </div>
            </div>
            <div class="row">
                <div class="col">
                    <p class="fw-bold">System packages</p>
                </div>
                <div class="col">
                    <p class="font-monospace">
                        {{.SystemInformation.SystemPackages | safe}}
                    </p>
                </div>
            </div>
            <div class="row">
                <div class="col">
                    <p class="fw-bold">R version</p>
                </div>
                <div class="col">
                    {{.SystemInformation.RVersion}}
                </div>
            </div>
            <div class="row">
                <div class="col">
                    <p class="fw-bold">Time</p>
                </div>
                <div class="col">
                    {{.SystemInformation.Time}}
                </div>
            </div>
            <div class="row">
                <div class="col">
                    <p class="fw-bold">Environment variables</p>
                </div>
                <div class="col">
                    <p class="font-monospace">
                        {{.SystemInformation.EnvVariables | safe}}
                    </p>
                </div>
            </div>
            <div class="row">
                <div class="col">
                    <p class="fw-bold">Hostname</p>
                </div>
                <div class="col">
                    {{.SystemInformation.Hostname}}
                </div>
            </div>
        </div>
    </div>

</body>

</html>
`
