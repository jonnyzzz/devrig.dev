We use `this_type_of_naming` for JSON and YAML fields.

Use this file to add or update the project's specifications. It is helpful for humans and AI. 

# Consistency and Security (TBD)
This section is in progress. Please create a PR with improvements to make it even more specific (and tested).


We design devrig to avoid potential attacks on the infrastructure, developer environments we create, and other cases. 

The devrig tool is installed in a customer's repository. We trust the customer project repository by default.
That allows the use of hash sums in the configuration scripts. 

We promote minimal trust; even this tool repository should not be trusted by default by the deployed tool binary. 

The user/developer of a customer's repository is responsible for double-checking that they are using the correct artifacts, hash sums, and signatures. 

All binaries that we download (IDEs, editors, plugins, extensions, settings) must be signed and/or include hash sums from the configuration in a customer repository. 

# Installation in a customer's repository (TBD)

devrig stores the following files in the customer's repository, to enable one-click development environment setup:
* `devrig/devrig.yaml` -- the configuration file in HCL or YAML
* `devrig.cmd` -- the entrypoint universal script which runs on all OS, including Windows, MacOS, and Linux

# Supported OS and CPUs

devrig is designed to work on Windows, Linux, and macOS. We support both x86-64 and ARM64 architectures. 
We do not support Intel Macs.

# Working with devrig

Once executed, the `devrig.cmd` script downloads, validates the signature/hashes, and caches the actual devrig binary
under the `devrig/.cache` folder. The cache path includes the OS name and CPU architecture to avoid clashes in multi-OS development.

The actual checksum must be included in the devrig configuration files to make sure that only approved binaries are used.

Checksums are created in a format that lists all OS and CPU architectures, to allow a one-line setup. 
The `devrig` tool provides helper commands to update the current version to the actual one; by doing so, it generates the patch to
the configuration file with the new URL and the new checksums. 

It is yet to be decided whether we should keep 6 URL checksum pairs in the configuration file or wrap them all into one line.

# Isolation of Configurations

By default, unless requested, we start IntelliJ and VSCode with isolated configuration/plugin/caches/logs folders. We make the
Clean the developer environment and make sure any local tools are not affecting the created environment. 

It feels like some settings, like keymaps, or some settings like colors/fonts, are still necessary to synchronize.
In a best-effort approach for now, and looking forward to receiving patches to support two-way replication.


# AI Tools

We want to make devrig to enable AI developer tools and agents. Please help us collect ideas in the project issues.

