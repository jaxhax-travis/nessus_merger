# Nessus Merger
A multi-threaded CLI tool written in Golang for merging multiple .nessus files into a single .nessus file.  This tool was inspired by the Python version fork found at https://github.com/elreydetoda/nessus_merger.

The Python version worked great but was slow when merging several large reports.  The goal of this project was to create something that accomplished that goal but faster.  I also just wanted an excuse to build something in Golang.  This is that.


## Installation
### Build from Source
This tool can be built from source using `go build`.

### Pre-compiled Binary
Addtionally, a Windows can be found under the [releases](https://github.com/jaxhax-travis/nessus_merger/releases) page.


## Usage
This tool will take the reports in a directory passed to in via the `-dir` or `--dir` argument and merge them into the file specified in the `-out` or `--out` filepath (you should remeber to include the `.nessus` extension if you want to import the report back into the web UI).

Optionally, you can provide a `-title` or `--title` argument with a string to provide a custom title on the merged report.  This is the text that the report will show up as when merged back into the web UI.  If this argument is not provided, it will use the default string of `Merged Report`.


## Help Screen

```
$ ./nessus_merger --help

        ---===[ Merge Nessus Reports v1.0 ]===---

Usage of ./nessus_merger:
  -dir string
        directory that contains existing .nessus files
  -out string
        Filepath you want to export the merged report to
  -title string
        The display name of merged report for the Nessus Web UI (default "Merged Report")
```

## How it Works
This application is multithread via Golang's GoRoutines.  There are two single use background GoRoutines.  One is for printing status updates on the jobs, and the other is a report merger GoRoutine.  There will also be one thread per report created that handles reading the `.nessus` files in and will unmarshal them and pass them over to the report merger GoRoutine.

The report merger GoRoutine will handle the reports as they come in.The first one will become the final report object, since there is no processing.  Each additional report that comes in will be processed and each host will reviewed.  If the hosts doesn't in the final report object, it will appended completely into the final report.  If the hosts does exist, it will iterate over the findings and check if the finding already exist in the final report by finding ID and port.  If so, it is skipped.  If not, then it is added.

Once there are no more reports, the final report file is written out.
