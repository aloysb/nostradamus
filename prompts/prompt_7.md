# Current task
The current task is simply a refactoring task: we want to ensure that we leave the code as readable and usable as possible.

You are forbidden to remove any code at this stage, this is purely a refactoring phase w here you will move across files to improve the maintainability of the script.

The code is located in a single file: cmd/main.go

You need to split it and organise across multiple file as shown in the "Expected structu re" below.

All files are already created. You simply need to move the functions/types around, add t he package declaration as neede and refer them in the import statement.

## Expected structure
The ideal output files would be organised as follow:

project/
├── cmd/
│   └── main.go                # Entry point, starts the application
│   └── main_test.go            # Main test file
├── internal/
│   ├── config/
│   │   └── config.go          # Handles environment variables and global settings
│   ├── logger/
│   │   └── logger.go          # Responsible for logging
│   ├── llm/
│   │   └── client.go          # LLM API client functions

│   │   └── critique.go        # Critique-related API client functions
│   └── models/
│       ├── prediction.go      # Models for predictions
│       └── critique.go        # Models for critiqued predictions
├── go.mod
└── go.sum

## Task detail

UPDATE `cmd/main.go`. You need to simply `cmd/main.go` so it is only an entry points. It contains a lot of type and function related to the llm/models packages.

