### Introduction to Rosters

Rosters is a git-native, JSONL-based issue tracker designed for AI agent workflows and developer teams. It operates directly on files within a `.rosters/` directory in your project, making it fast, transparent, and merge-friendly. It has zero runtime dependencies besides Bun.

### Global Flags

These flags can be used with most `rt` commands:

| Flag | Alias | Description |
| :--- | :--- | :--- |
| `--json` | | Output results in a machine-readable JSON format. It's an alias for `--format json`. |
| `--format <mode>` | | Specifies the output format. Modes include `markdown` (default, colored), `compact` (terse, one-line), `plain` (no colors), `ids` (only issue IDs, one per line), and `json`. |
| `--quiet` | `-q` | Suppresses all non-error output, useful for scripting. |
| `--verbose` | | Enables extra diagnostic output. |
| `--timing` | | Prints the command's execution time to `stderr`. |
| `--version`| `-v` | Prints the installed version of `rt`. |
| `--help` | `-h` | Displays help information for a command. |

---

### Command Reference

Commands are grouped by their primary function.

#### Project Initialization & Health

##### `rt init`
Initializes a new rosters repository in the current directory.

-   Description: This is the first command you should run in a new project. It creates the `.rosters/` directory and all necessary files (`config.yaml`, `issues.jsonl`, `templates.jsonl`, `plans.jsonl`, `.gitignore`). It also appends a `merge=union` strategy for JSONL files to the project's `.gitattributes`, which helps resolve merge conflicts automatically.
-   Example:
    ```bash
    # Set up rosters for the current project
    rt init
    ```

##### `rt doctor`
Checks the health and integrity of the `.rosters/` repository.

-   Description: Validates everything from file integrity and data schema to dependency consistency and stale locks. It's a useful tool for diagnosing problems.
-   Options:
    -   `--fix`: Automatically fixes any issues that are safely correctable.
-   Examples:
    ```bash
    # Check for any issues
    rt doctor

    # Find and automatically fix problems
    rt doctor --fix
    ```

---

#### Issue Management (CRUD)

##### `rt create`
Creates a new issue.

-   Options:
    -   `--title <text>`: (Required) The title of the issue.
    -   `--type <type>`: The type of issue. Can be `task`, `bug`, `feature`, or `epic`. Defaults to `task`.
    -   `--priority <n>`: Sets the priority from `0` (Critical) to `4` (Backlog). Accepts `P0`-`P4` notation. Defaults to `2` (Medium).
    -   `--description <text>`: A longer description of the issue. Aliases: `--desc`, `--body`.
    -   `--assignee <name>`: Assigns the issue to a user or agent.
    -   `--labels <labels>`: A comma-separated list of labels to add to the issue.
-   Example:
    ```bash
    # Create a high-priority bug with a description and assignee
    rt create --title "Authentication fails with special characters" \
               --type bug \
               --priority P1 \
               --description "Users with '&' or '#' in their password cannot log in." \
               --assignee "builder-1" \
               --labels "auth,bug"
    ```

##### `rt show <id> [<id2> ...]`
Displays detailed information for one or more issues or plans.

-   Description: When given a single issue ID, it shows full details. When given multiple IDs, it lists the details for each, separated by a divider. If an ID starts with `pl-`, it routes to `rt plan show`.
-   Example:
    ```bash
    # Show details for a single issue
    rt show myproject-a1b2

    # Show details for multiple issues at once
    rt show myproject-a1b2 myproject-c3d4
    ```
##### `rt list`
Lists issues with powerful filtering and sorting capabilities.

-   Description: By default, it lists `open` and `in_progress` issues, sorted by priority.
-   Filter Options:
    -   `--status <status>`: Filter by `open`, `in_progress`, or `closed`.
    -   `--type <type>`: Filter by `task`, `bug`, etc.
    -   `--assignee <name>`: Filter by assignee.
    -   `--label <labels>`: Filter for issues having ALL specified labels (comma-separated).
    -   `--label-any <labels>`: Filter for issues having ANY of the specified labels.
    -   `--unlabeled`: Show only issues with no labels.
    -   `--priority <levels>`: Filter by an exact set of priority levels (e.g., `0,1` or `P0,P1`).
    -   `--priority-max <n>`: Filter for issues with priority at or below `n`.
    -   `--all`: Includes `closed` issues in the results.
    -   `--limit <n>`: Limits the number of issues shown.
-   Sort Options:
    -   `--sort <mode>`: Sorts the output. Modes are `priority` (default), `created`, `updated`, `id`.
-   Examples:
    ```bash
    # List all open, high-priority bugs assigned to "alice"
    rt list --status open --type bug --priority-max 1 --assignee alice

    # List the 10 most recently created issues with the "ui" label
    rt list --label ui --sort created --limit 10
    ```
##### `rt search <query>`
Performs a case-insensitive text search on issue titles and descriptions.

-   Description: Includes closed issues by default. It accepts the same filtering and sorting flags as `rt list`.
-   Example:
    ```bash
    # Search for "database connection" in all open tasks and bugs
    rt search "database connection" --status open --type task,bug
    ```
##### `rt update <id>`
Modifies the fields of an existing issue.

-   Description: Use this to change an issue's status, assignee, title, etc.
-   Options:
    -   `--status <status>`: Change status (e.g., to `in_progress` to claim it).
    -   `--title <text>`: Change the title.
    -   `--assignee <name>`: Change the assignee.
    -   `--description <text>`: Change the description.
    -   `--type <type>`: Change the type.
    -   `--priority <n>`: Change the priority.
    -   `--add-label <labels>`: Adds one or more comma-separated labels.
    -   `--remove-label <labels>`: Removes one or more comma-separated labels.
    -   `--set-labels <labels>`: Replaces all existing labels with the new set. An empty string clears all labels.
    -   `--extensions <json>`: Shallow-merges a JSON object into the `extensions` field for runtime metadata.
    -   `--clear-extensions`: Removes the `extensions` field entirely.
-   Example:
    ```bash
    # Claim an issue and raise its priority
    rt update myproject-a1b2 --status in_progress --priority 1

    # Add a label and update the title
    rt update myproject-c3d4 --add-label "backend" --title "API: Fix connection pooling"
    ```
##### `rt close <id> [<id2> ...]`
Closes one or more issues.

-   Options:
    -   `--reason <text>`: Adds a reason for closing the issue.
-   Example:
    ```bash
    # Close an issue with a reason
    rt close myproject-a1b2 --reason "Completed in PR #123"

    # Close multiple issues at once
    rt close myproject-c3d4 myproject-e5f6
    ```
##### `rt dep`
Manages dependencies between issues (A depends on B).

-   Subcommands:
    -   `add <issue> <depends-on>`: Makes `<issue>` depend on `<depends-on>`.
    -   `remove <issue> <depends-on>`: Removes a dependency.
    -   `list <issue>`: Shows all dependencies for an issue.
-   Example:
    ```bash
    # Make issue-B depend on issue-A
    rt dep add issue-B issue-A
    ```

