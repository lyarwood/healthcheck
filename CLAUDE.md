# Claude Instructions for KubeVirt Healthcheck Project

## Commit Guidelines

When creating commits for this project, always include an assisted-by line attributing Claude:

```
Assisted-By: Claude <noreply@anthropic.com>
```

This should be included at the end of all commit messages:

```
Assisted-By: Claude <noreply@anthropic.com>
```

### Commit Message Format

- Keep commit messages wrapped at 80 characters per line
- Use a descriptive subject line (50 characters or less)
- Add a blank line between subject and body
- Wrap the body text at 80 characters

## Project Context

This is a Go CLI tool for parsing KubeVirt CI health data and reporting failed tests. The tool helps developers and maintainers analyze test failures from the kubevirt/ci-health project by providing various filtering and grouping options.

## Testing Commands

When making changes, ensure the tool builds and runs correctly:

```bash
go build
./healthcheck --help
```

Test different functionality modes:
- `./healthcheck merge -j compute` - Filter by job regex
- `./healthcheck merge -c` - Count failures
- `./healthcheck merge --lane-run` - Group by lane run UUID
- `./healthcheck merge -f` - Show failure details