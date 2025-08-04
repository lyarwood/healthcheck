# Claude Instructions for KubeVirt Healthcheck Project

## Commit Guidelines

When creating commits for this project, always include a co-authored-by line attributing Claude:

```
Co-Authored-By: Claude <noreply@anthropic.com>
```

This should be included at the end of all commit messages, along with the standard Claude Code attribution:

```
🤖 Generated with [Claude Code](https://claude.ai/code)

Co-Authored-By: Claude <noreply@anthropic.com>
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
- `./healthcheck -j compute` - Filter by job regex
- `./healthcheck -c` - Count failures
- `./healthcheck --lane-run` - Group by lane run UUID
- `./healthcheck -f` - Show failure details