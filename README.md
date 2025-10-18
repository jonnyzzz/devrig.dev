# devrig.dev
[![Apache 2.0 License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![Build](https://github.com/jonnyzzz/devrig.dev/actions/workflows/go.yml/badge.svg)](https://github.com/jonnyzzz/devrig.dev/actions/workflows/go.yml)

The command to start the AI-empowered development environment of your project.
With all possible IDEs, extensions, and configurations.
With local development, Docker, and remote development.
Including IntelliJ, Cursor, VSCode, and any other IDEs.

## Goal

Create the world's most popular universal tool to start a pre-configured, fine-tailored development environment for humans and agents.
We make new developers and agents ready for coding after only one command: `./idew start`.

## What is devrig?

IDE Wrapper is a universal command-line tool that automatically downloads, configures, and launches pre-configured development environments for your projects.
One command (`./idew start`) gets any developer or AI agent ready to code with the exact IDE setup their project needs.

**Problem it solves:** No more "it works on my machine" issues. No manual IDE installation, extension hunting, or configuration copying. Perfect for:
- Onboarding new developers in minutes
- AI coding agents needing consistent environments  
- Open-source projects with complex setup requirements
- Teams wanting standardized development environments
- Easy transition to remote development or docker environments

## Project Status

⚠️ **Alpha/Proof of Concept** - This project is in early development. Everything can change.

The tool looks for the `.idew.yaml` file for configuration. 

The IDE is downloaded and configured under the `.idew` directory next to the `.idew.yaml` file. 

The file is of the following format, which allows specifying the IDE that is used to open the project, e.g. 

```yaml
wrapper:
    version: 0.0.1
    hash: sha-512:<hash>

ide:
    name: GoLand        #IDE which is needed
    version: 2024.3     #public version
    hash: sha-512:<hash>
    #build: optional build number
    #more ide-spefic parameters 
```

The tool is compatible with [devcontainer](https://containers.dev/) to provide configurations.

# Contribute

We welcome contributions to the IDE Wrapper project! Here are some ways you can contribute:

1. **Report Bugs**: If you encounter any issues, please report them on our [issue tracker](https://github.com/jonnyzzz/ide-wrapper/issues).

2. **Suggest Features**: Have an idea for a new feature? Let us know by opening a feature request on our [issue tracker](https://github.com/jonnyzzz/ide-wrapper/issues).

3. **Submit Pull Requests**: If you have a fix or a new feature, feel free to submit a pull request. Please make sure to follow our [contribution guidelines](https://github.com/jonnyzzz/ide-wrapper/CONTRIBUTING.md).

4. **Improve Documentation**: Help us improve our documentation by making it clearer and more comprehensive.

Thank you for your contributions!
