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


# AI Coding Agents to support next

Here is an expanded and structured list of AI coding agents you can install and run locally. The ranking is approximate, based on recent community popularity, GitHub stars, expert reviews, and widespread use. "Can support offline mode" means the agent can function fully on your hardware, without sending code or prompts to third-party servers.

Perfect! Here's the complete merged list of **31 AI coding agents** with all information consolidated:

| Name               | Agent URL                                        | Rank | Offline Mode Support |
|--------------------|--------------------------------------------------|------|----------------------|
| Tabby              | https://github.com/TabbyML/tabby                 | 1    | Can                  |
| Continue           | https://github.com/continuedev/continue          | 2    | Can                  |
| Cline              | https://github.com/ClineDev/Cline                | 3    | Can                  |
| ForgeCode          | https://github.com/ForgeAI-Dev/ForgeCode         | 4    | Can                  |
| Aider              | https://github.com/paul-gauthier/aider           | 5    | Can                  |
| Zed                | https://zed.dev/                                 | 6    | Can                  |
| Observer           | https://github.com/Roy3838/Observer              | 7    | Can                  |
| CodeGenie          | https://github.com/sherinjosephroy/codegenie     | 8    | Can                  |
| Goose              | https://github.com/abacaj/goose                  | 9    | Can                  |
| Gemini CLI         | https://github.com/google/gemini-cli             | 10   | Can                  |
| Codestral          | https://github.com/numind-ai/codestral           | 11   | Can                  |
| Qwen Coder         | https://github.com/QwenLM/Qwen1.5-7B-Chat        | 12   | Can                  |
| Opencode           | https://opencode.com/                            | 13   | Can                  |
| OpenHands          | https://github.com/All-Hands-AI/OpenHands        | 14   | Can                  |
| Cherry Studio      | https://github.com/CherryHQ/cherry-studio        | 15   | Can                  |
| StarCoder          | https://huggingface.co/bigcode/starcoder         | 16   | Can                  |
| FastGPT            | https://github.com/labring/FastGPT               | 17   | Can                  |
| MetaGPT            | https://github.com/geekan/MetaGPT                | 18   | Can                  |
| Kimi CLI           | https://github.com/moonshotai/kimi-cli           | 19   | Can                  |
| Open Interpreter   | https://github.com/KillianLucas/open-interpreter | 20   | Can                  |
| Claude Code        | https://claude.com/cli                           | 21   | Can                  |
| Codex CLI          | https://github.com/openai/codex                  | 22   | Can                  |
| Cursor             | https://cursor.so/                               | 23   | Can                  |
| Copilot (CLI)      | https://github.com/github/copilot-cli            | 24   | Cannot               |
| Gemini Code Assist | https://cloud.google.com/ai/gemini/code-assist   | 25   | Cannot               |
| Windsurf           | https://github.com/codeium/windsurf              | 26   | Can                  |
| Bolt.new           | https://github.com/boltai/bolt.new               | 27   | Can                  |
| CodeGPT            | https://codegpt.co/                              | 28   | Can                  |
| Replit Agent       | https://replit.com/site/ai                       | 29   | Cannot               |
| Postman AI         | https://www.postman.com/ai                       | 30   | Cannot               |
| Snyk DeepCode      | https://snyk.io/deepcode/                        | 31   | Can                  |
| Ona                |                                                  | 32   | ?                    |

