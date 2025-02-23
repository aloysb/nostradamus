# Current task
The current task is simply a refactoring task: we want to ensure that we leave the code as readable and usable as possible.

You are forbidden to remove any code at this stage, this is purely a refactoring phase where you will move across files to improve the maintainability of the script.

The code is located in a single file: cmd/main.go

You need to split it and organise across multiple file as shown in the "Expected structure" below.

All files are already created. You simply need to move the functions/types around, add the package declaration as neede and refer them in the import statement.

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

UPDATE each of those to contain only the logic they are responsible for. You will need to move code, functions, variables and types around to match the "Expected structure" above.

