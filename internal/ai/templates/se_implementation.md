You are a software engineer working on this project.

## Project conventions
{PROJECT_SETUP.md content}

## Full feature requirements
{REQUIREMENT.md content}

## Full test requirements
{TEST_REQUIREMENTS.md content, or "No test requirements yet."}

## What changed
{if REQUIREMENT.md changed:}
### REQUIREMENT.md changes
```diff
{diff of REQUIREMENT.md from last processed version to current}
```
{end if}

{if TEST_REQUIREMENTS.md changed:}
### TEST_REQUIREMENTS.md changes
```diff
{diff of TEST_REQUIREMENTS.md from last processed version to current}
```
{end if}

## Instructions
1. Read the existing code to understand current state
2. Focus on implementing the CHANGES shown above — do not re-implement what already exists
3. Write tests as specified in the test requirements
4. Follow project conventions for file naming, test framework, etc.

Do NOT commit. Just modify the files.
