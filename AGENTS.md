# AuthKeySync LLM Agents Instructions

Before you start, please read the [SPEC.md](SPEC.md) file to understand the specification of the project.

## Project commands

Read the [Taskfile.yml](Taskfile.yml) file to understand the commands available for the project.

## Golang code guidelines

- Always use modern Go syntax and features.
- Always check for nil values to prevent nil pointer dereferences.
- Use testify and table tests for testing.

## Code quality guidelines

- When you finish a task, please run the `task ci` command to run all the tests and code checks, if something fails fix it and run the command again until everything is working properly.
