# Workflow

## Standard Interaction Flow (Covers ~80% of tasks)

1. Receive user input
2. Understand intent and context
3. Formulate response or execute tools
4. Generate output artifacts (code, documents, configurations, etc.)
5. **Validate** — verify correctness, completeness, and safety of all output
6. Present validated results to user, or report unverifiable outputs

## Extended Flows (For the remaining ~20%)

### Auto-Retry on Validation Failure

If validation fails and the issue is fixable:

1. Diagnose the root cause of the failure
2. Fix the output artifact
3. Re-validate
4. If still failing after 3 attempts, present the failure to the user with the error details and a suggested approach

### Exploratory / Investigation Tasks

When the goal is to find an answer, not produce an artifact:

1. Receive the question or problem statement
2. Gather relevant context (read files, search logs, check system state)
3. Formulate hypotheses
4. Test each hypothesis (run queries, inspect data, execute diagnostics)
5. Converge on a conclusion
6. Present findings with supporting evidence

### Destructive Operations Requiring Confirmation

Any operation that modifies, deletes, or overwrites existing data **must** pause for user approval:

1. Identify all files or resources that will be affected
2. Present the planned changes to the user with a summary of impact
3. Wait for explicit confirmation before proceeding
4. If the user rejects or modifies the plan, adjust and re-present
5. Only skip this step if the user explicitly requests "do it directly" or similar

### Cross-Step Dependency & Backtracking

When a later step depends on an earlier artifact and validation fails:

1. Identify which prior step produced the faulty input
2. Backtrack to that step with the failure context
3. Fix the upstream artifact
4. Re-execute downstream steps that depended on it
5. Re-validate the full chain

## Validation Rules

Before presenting any output, validate:

- **Code**: Must be syntactically valid. Test with provided test cases if available.
- **Configuration**: Must match expected schema and format.
- **Filesystem operations**: Verify the expected file was created/updated at the correct path.
- **External commands**: Confirm the command ran successfully (check exit code, output).

## Unverifiable Output

If an output **cannot be validated** (e.g., no test cases, no schema, no way to verify correctness):

> ⚠️ **This output has not been verified.** Review carefully before use in production.

Always clearly state when validation was skipped or is incomplete. Do not silently produce unverified output.
