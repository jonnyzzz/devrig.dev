# IDE Wrapper

A command-line tool that helps download, install, manage, and configure IDE and development environments for your project. 
Inspired by the simplicity of Gradle and Maven Wrappers, this tool enables a straightforward setup for both new and experienced
contributors—simply include the binary and a config file in your repository, and you’re good to go!

# License

This project is licensed under the Apache 2.0 License - see the [LICENSE](LICENSE) file for details.

[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

## Goal

We are building the universal tool to help configuring and opening
projects in IDEs. 

## Proof of Concept

The tool looks for the `.ides.yaml` file for configuration. 

The IDE is downloaded and configured under the `.ides` directory next to the `.ides.yaml` file. 

The file is of the following format, it allows to specify the IDE which is used to open the project, e.g. 

```yaml
ide:
    name: GoLand        #IDE which is needed
    version: 2024.3     #public version
    #build: optional build number
    #more ide-spefic parameters 
```

# Contribute

We welcome contributions to the IDE Wrapper project! Here are some ways you can contribute:

1. **Report Bugs**: If you encounter any issues, please report them on our [issue tracker](https://github.com/jonnyzzz/ide-wrapper/issues).

2. **Suggest Features**: Have an idea for a new feature? Let us know by opening a feature request on our [issue tracker](https://github.com/jonnyzzz/ide-wrapper/issues).

3. **Submit Pull Requests**: If you have a fix or a new feature, feel free to submit a pull request. Please make sure to follow our [contribution guidelines](https://github.com/jonnyzzz/ide-wrapper/CONTRIBUTING.md).

4. **Improve Documentation**: Help us improve our documentation by making it clearer and more comprehensive.

Thank you for your contributions!