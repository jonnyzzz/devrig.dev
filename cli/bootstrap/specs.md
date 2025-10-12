# Bootstrap contains the scripts to implement the entry-point to the devrig. 

## Definitions

### devrig.yaml

**devrig.yaml** -- the main config file of the `devrig` tool.
It contains the URLs and hash sums for the binaries for all 5 options (3 OS, 2 CPU types), the IDEs,
and other configuration options of the developer environment.

The location of the `devrig.yaml` is the same as the location of the bootstrap script(s).
The `devrig.yaml` location can be overridden with `DEVRIG_CONFIG` environment variable (must be clearly logged to the console).

In the documents below, we simply say `devrig.yaml` and refer to this definition and ability to override the file location.

### .devrig folder or devrig home

**.devrig folder** -- the folder, where the binaries are stored. It is `.devrig` folder in the location of the bootstrap script(s).
The `devrig home` location can be overridden with `DEVRIG_HOME` environment variable (must be clearly logged to the console).

In the documents below, we simply say __`.devrig` folder__ and refer to this definition and ability to override the folder location.


# .devrig folder layout

We support various versions, OS, and CPU type. The layout allows re-using the same working directory
for all OS and CPU types at the same time, having multiple versions of the binaries on the disk. 

The layout is as follows:
`.devrig/<tool-name>-<os>-<cpu-type>-<version?><hash><modifier?>/tool contents`

The `modifier` is optional and used to keep temporary files under the folder. It starts with `-` if present.
The `version` is optional and if present ends with `-`.

# The bootstrap Logic

## The logic requirements
- it supports Windows, Linux, and macOS
- it supports ARM64 and x86-64 (we do not support Intel Macs)
- it has minimal dependencies (no need to install any other tools)
- it is covered with integration tests

# How it works
- In the YAML, there is `devrig` section, with binaries and hash sums for all 5 options (3 OS, 2 CPU types)
- The main login of the bootstrap script: it takes any commandline parameters and passes them to the `devrig` binary
- How it works:
  - it reads the `devrig.yaml` config to get the url and hash sum for the binary for the current OS and CPU type
  - it checks if the binary is present under the `.devrig` folder next to the script location
  - it validates the hash sum of the binary against the hash sum from the `devrig.yaml`
  - the script fails with error if the checksum does not match
  - it executes the binary with the passed parameters and environment variables
  - if the binary is not present, it downloads the binary from the URL given
  - it stores the binary to a temporary name in the `.devrig` folder, following the layout described above
  - it checks the hash sum of the downloaded binary against the hash sum from the `devrig.yaml`
  - if the hash sum is not matching, the error is shown and the tool exists with error message
  - the downloaded and validated binary is moved to the production location under the `.devrig` folder
  - the tool restarts to apply the happy path above and to execute the tool.

# Bootstrap Loop

The bootstrap script is executed to download a functional `devrig` binary.

The binary supports the `./devrig init <target folder>` command to create the `.dervig`
environment for the new project.

It supports the `./devrig version <command>` (or `--v`) command to
show the current version of the `devrig` binary and to check for updates.

It supports the `./devrig update <command>` to update the current `devrig` binary
to the new version, the update changes the `devrig.yaml` file to include new
version of the tool, version, download URL, and hash sums. 

## How the update works

We encode public keys into each `devrig` release, which are needed to verify
the original updates from the `https://devrig.dev/download/latest.json` file.
It is done to lower the probability of an upgrade to a malicious version.

The `latest.json` file contains the list of all recent releases, release notes, versions.
We keep signature for that file in the `https://devrig.dev/download/latest.json.sig` file.
We believe that `sha512` is enough to protect the actual binaries, so we only sign
the sha512 signatures inside the `latest.json` file.`

The update process can potentially change the code of the `devrig` scripts in the
root of the repository. We bundle these scripts into the actual `devrig` binary to allow
the `init` command to work without any additional dependencies.

