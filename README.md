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
##### `rt block` / `unblock`
A more intuitive way to manage dependencies.

-   Commands:
    -   `block <id> --by <blocker-id>`: Marks `<id>` as blocked by `<blocker-id>`. (Equivalent to `rt dep add <id> <blocker-id>`).
    -   `unblock <id> --from <blocker-id>`: Removes a specific blocker.
    -   `unblock <id> --all`: Removes all *closed* blockers from an issue.
-   Example:
    ```bash
    # Mark issue-B as blocked by issue-A
    rt block issue-B --by issue-A

    # Remove the block
    rt unblock issue-B --from issue-A
    ```

##### `rt blocked`
Lists all issues that are currently blocked by at least one open issue.

##### `rt ready`
Shows all `open` issues that have no open blockers. This is the primary command for finding available work. It supports the same filter and sort flags as `rt list`.

-   Options:
    -   `--respect-schedule`: An opt-in flag that excludes issues parked via the `extensions` field (where `extensions.queued === true` or `extensions.scheduledFor` is in the future).
-   Example:
    ```bash
    # Find the highest-priority work available for the "backend" team
    rt ready --label backend --sort priority
    ```

##### `rt label`
A dedicated command group for managing labels.

-   Subcommands:
    -   `add <id> <label...>`: Adds one or more labels to an issue.
    -   `remove <id> <label...>`: Removes one or more labels from an issue.
    -   `list <id>`: Lists all labels on a specific issue.
    -   `list-all`: Lists all unique labels used across the entire project, with counts.
-   Example:
    ```bash
    # Add two labels to an issue
    rt label add myproject-a1b2 backend performance

    # See all labels used in the project
    rt label list-all
    ```
#### Structured Planning (`rt plan`)

This command tree facilitates lreaking down large or ambiguous work into smaller, manageable child issues.

##### `rt plan templates`
Lists available plan templates (e.g., `feature`, `bug`, `refactor`).

##### `rt plan prompt <roster-id>`
Generates a structured JSON prompt for an LLM to fill out, based on a template.

-   Description: This is the first step in the planning workflow. The output is a JSON object describing the sections the LLM needs to complete.
-   Options:
    -   `--template <name>`: Overrides the template automatically inferred from the roster's type.
    -   `--domain <name>`: (Lore integration) Forces the domain for enriching the prompt with prior art.

##### `rt plan submit <roster-id>`
Validates a completed plan JSON, spawns child rosters for each step, and links them.

-   Options:
    -   `--plan <file>`: (Required) Path to the plan JSON file. Use `-` to read from stdin.
    -   `--name <text>`: Sets a human-readable name for the plan.
    -   `--overwrite`: Replaces an existing plan for the roster, preserving child issue IDs where possible and flagging obsolete ones.
    -   `--record-decision`: (Lore integration) After a successful submission, records the plan's `approach` section as a decision in Lore.
    -   `--domain <name>`: (Lore integration) Forces the domain for `--record-decision`.

##### `rt plan adopt <plan-id> <roster-id...>`
Links existing open rosters into a plan. This is a link-only operation and does not change the adopted roster's status, title, or other fields.

-   Options:
    -   `--step <i>`: Anchors the adopted roster to a specific 1-based step in the plan's blueprint.

##### `rt plan release <plan-id> <roster-id...>`
Detaches rosters from a plan without closing them. This is the inverse of `adopt`.

##### `rt plan show <pl-id|roster-id>`
Displays a plan's sections, child status, and any nested sub-plans.

##### `rt plan list`
Lists all plans, with filtering options like `--roster`, `--status`, and `--template`.

##### `rt plan validate <pl-id|roster-id>`
Re-runs validation for a stored plan against its template definition.

##### `rt plan outcome <pl-id|roster-id>`
Records the outcome of a plan. This is for informational purposes and doesn't affect issue status.

-   Options:
    -   `--result <value>`: (Required) `success`, `partial`, or `failure`.
    -   `--note <text>`: An optional note explaining the outcome.

##### `rt plan review <pl-id|roster-id>`
Records who reviewed a plan. This is for informational purposes.

-   Options:
    -   `--by <name>`: (Required) The name of the reviewer.

-   Example Planning Workflow:
    ```bash
    # 1. Get a structured prompt for a large feature
    rt plan prompt myproject-e5f6 --json > plan_request.json

    # (An LLM or user fills out plan_request.json to create plan.json)

    # 2. Submit the completed plan
    rt plan submit myproject-e5f6 --plan plan.json --name "User Authentication Flow"

    # 3. View the created plan
    rt plan show pl-a1b2

    # 4. Adopt an existing, related bug into the plan
    rt plan adopt pl-a1b2 myproject-c3d4 --step 3
    ```

#### Templates/Molecules (`rt tpl`)

For creating repeatable sequences of issues.

-   Subcommands:
    -   `create --name <text>`: Creates a new template.
    -   `step add <id> --title <text>`: Adds a step to a template. The title can include `{prefix}`.
    -   `list`: Lists all templates.
    -   `show <id>`: Shows a template's details and steps.
    -   `pour <id> --prefix <text>`: Instantiates a template, creating a chain of dependent issues. The prefix replaces `{prefix}` in step titles.
    -   `status <id>`: Shows the completion status of a "poured" template instance.

