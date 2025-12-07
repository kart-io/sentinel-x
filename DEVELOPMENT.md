# Development Guide

This project follows a Kubernetes-style Monorepo architecture. This document explains how to work with the codebase, manage dependencies, and contribute to the core `goagent` module.

## Project Structure

*   **`staging/src/github.com/kart-io/goagent`**: This is the **Source of Truth** for the `goagent` library. It is a regular directory within this repository, not a submodule. You should develop and modify `goagent` code directly here.
*   **`vendor/`**: Contains all project dependencies (including third-party libraries). This directory is **committed to Git** to ensure hermetic builds.
*   **`go.mod`**: Uses a `replace` directive to point `github.com/kart-io/goagent` to the local `./staging/src/github.com/kart-io/goagent` directory.

## Development Workflow

### 1. Modifying `goagent`

Since `goagent` is part of the monorepo, you can modify its code directly in `staging/src/github.com/kart-io/goagent`.

*   **No `go get` required**: Your changes are immediately reflected in `sentinel-x` due to the local `replace` directive.
*   **Atomic Commits**: You can commit changes to both `sentinel-x` (application) and `goagent` (library) in a single Git commit.

### 2. Running Tests

*   **Run all tests (sentinel-x)**:
    ```bash
    go test ./...
    ```
    *Note: This runs tests for the main module packages.*

*   **Run goagent tests**:
    Because `goagent` is technically a separate module (replaced locally), you must run its tests from its directory:
    ```bash
    cd staging/src/github.com/kart-io/goagent && go test ./...
    ```

### 3. Syncing with Upstream `goagent`

If the standalone `goagent` repository (`github.com/kart-io/goagent`) has been updated by others, you can pull those changes into this monorepo.

**Warning**: This will overwrite any uncommitted local changes in `staging/`.

```bash
make update-goagent
```

This script will:
1.  Clone the remote `goagent` repository.
2.  Replace the content of `staging/src/github.com/kart-io/goagent` with the remote content.
3.  Run `go mod tidy` and `go mod vendor`.

### 4. Publishing Changes to Upstream `goagent`

If you have made changes to `goagent` locally and want to push them to the standalone repository (e.g., for other community users), use the publish script.

**Prerequisite**: You must have SSH push access to `git@github.com:kart-io/goagent.git`.

```bash
make publish-staging
```

This script will:
1.  Clone the remote `goagent` repository to a temp folder.
2.  Copy your local `staging/` content to it.
3.  Create a sync commit.
4.  Push to the `master` branch of the remote repository.

## Dependency Management

We verify dependencies into the repository.

*   **Adding a new dependency**:
    ```bash
    go get github.com/some/lib
    go mod vendor
    ```
*   **Updating dependencies**:
    ```bash
    go get -u github.com/some/lib
    go mod vendor
    ```

Always commit the changes to `go.mod`, `go.sum`, and the `vendor/` directory.
