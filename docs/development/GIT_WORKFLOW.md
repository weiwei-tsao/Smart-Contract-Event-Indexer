# Git Workflow & Commit Strategy

## Overview

This document defines the git workflow and commit strategy for the Smart Contract Event Indexer project. It emphasizes atomic commits for sub-tasks within phases to maintain clean history and enable better code review.

## Core Principles

### 1. Atomic Commits
- **One logical change per commit**
- **Each sub-task gets its own commit**
- **Commits should be reviewable independently**
- **Easy to revert individual changes if needed**

### 2. Conventional Commits
- **Format**: `type(scope): description`
- **Types**: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`
- **Scope**: Service or component name (e.g., `api-gateway`, `query-service`)
- **Description**: Clear, concise description of the change

### 3. Phase-Based Development
- **Each phase has multiple sub-tasks**
- **Each sub-task should be committed when complete**
- **Use feature branches for phases**: `feature/phase-X-description`
- **Merge to main when phase is complete**

## Commit Strategy by Phase

### Phase Development Pattern

```
Phase 3: API Layer Development
├── Task 1: GraphQL Schema Design
│   ├── feat(graphql): design complete GraphQL schema with custom scalars
│   └── feat(graphql): configure gqlgen code generation
├── Task 2: gRPC Service Definitions
│   ├── feat(grpc): define QueryService proto interface
│   └── feat(grpc): define AdminService proto interface
├── Task 3: Query Service Implementation
│   ├── feat(query-service): implement gRPC server with interceptors
│   ├── feat(query-service): add Redis caching layer
│   ├── feat(query-service): build SQL query optimizer
│   └── feat(query-service): add Prometheus metrics
└── Task 4: API Gateway Implementation
    ├── feat(api-gateway): implement REST API endpoints
    ├── feat(api-gateway): add middleware for CORS and logging
    └── feat(api-gateway): implement health check endpoints
```

## Detailed Commit Guidelines

### 1. Commit Message Format

```bash
type(scope): brief description

Detailed description of what was implemented:
- Key changes made
- Files affected
- Dependencies added
- Configuration updated

Resolves: Phase X Task Y - Task Description
```

### 2. Commit Types

| Type | Description | Example |
|------|-------------|---------|
| `feat` | New feature or functionality | `feat(api-gateway): add event filtering endpoints` |
| `fix` | Bug fix | `fix(query-service): correct cache invalidation logic` |
| `docs` | Documentation changes | `docs(api): update GraphQL schema documentation` |
| `style` | Code style changes (formatting, etc.) | `style(query-service): format SQL queries` |
| `refactor` | Code refactoring without behavior change | `refactor(cache): extract cache key generation` |
| `test` | Adding or updating tests | `test(api-gateway): add integration tests for events` |
| `chore` | Maintenance tasks, dependencies | `chore(deps): update Go modules and dependencies` |

### 3. Scope Guidelines

- **Service names**: `api-gateway`, `query-service`, `admin-service`, `indexer-service`
- **Shared components**: `shared`, `proto`, `models`, `utils`
- **Infrastructure**: `docker`, `k8s`, `terraform`
- **Documentation**: `docs`, `api`, `architecture`

### 4. Sub-Task Commit Examples

#### GraphQL Schema Development
```bash
# Task 1.1: Basic schema structure
feat(graphql): design core GraphQL types and queries

- Add Event, Contract, EventArg type definitions
- Implement cursor-based pagination with EventConnection
- Define Query operations for events and contracts
- Add custom scalars for DateTime, BigInt, Address

Resolves: Phase 3 Task 1.1 - GraphQL Core Types

# Task 1.2: Custom scalars
feat(graphql): implement custom scalar serialization

- Add MarshalDateTime, UnmarshalDateTime functions
- Implement BigInt string conversion for precision
- Add Address validation and serialization
- Configure gqlgen scalar mappings

Resolves: Phase 3 Task 1.2 - Custom Scalar Implementation
```

#### Query Service Development
```bash
# Task 3.1: gRPC server
feat(query-service): implement gRPC server with interceptors

- Add QueryServiceServer with logging and metrics
- Implement unary and stream interceptors
- Add health check endpoint
- Configure Prometheus metrics collection

Resolves: Phase 3 Task 3.1 - gRPC Server Setup

# Task 3.2: Cache layer
feat(query-service): add Redis caching layer with TTL strategies

- Implement CacheManager with Redis client
- Add cache key generation and hashing
- Configure TTL strategies for different query types
- Add cache invalidation on contract updates

Resolves: Phase 3 Task 3.2 - Redis Caching Layer
```

## Branch Strategy

### 1. Branch Naming
- **Feature branches**: `feature/phase-X-description`
- **Bug fixes**: `fix/description`
- **Hotfixes**: `hotfix/description`
- **Documentation**: `docs/description`

### 2. Branch Lifecycle
```bash
# Start new phase
git checkout -b feature/phase-3-api-layer

# Work on sub-tasks with atomic commits
git add services/api-gateway/
git commit -m "feat(api-gateway): implement REST API endpoints"

git add services/query-service/
git commit -m "feat(query-service): add Redis caching layer"

# Continue with more sub-tasks...

# When phase is complete, merge to main
git checkout main
git merge feature/phase-3-api-layer
git tag v1.0.0-phase3
```

## Code Review Guidelines

### 1. Review Focus
- **Each commit should be reviewable independently**
- **Focus on one logical change per review**
- **Verify commit message clarity and accuracy**
- **Check that tests are included where appropriate**

### 2. Review Process
```bash
# Review individual commits
git log --oneline feature/phase-3-api-layer

# Review specific commit
git show <commit-hash>

# Review changes between commits
git diff <commit1> <commit2>
```

## Best Practices

### 1. Before Committing
- [ ] Run tests: `make test`
- [ ] Run linter: `make lint`
- [ ] Check git status: `git status`
- [ ] Review changes: `git diff --cached`

### 2. Commit Message Quality
- [ ] Use imperative mood ("add" not "added")
- [ ] Keep first line under 50 characters
- [ ] Include detailed description for complex changes
- [ ] Reference the specific task being resolved

### 3. Atomic Commit Checklist
- [ ] Does this commit represent one logical change?
- [ ] Can this commit be reverted independently?
- [ ] Is the commit message clear and descriptive?
- [ ] Are all related files included in this commit?
- [ ] Does the commit include necessary tests?

## Common Anti-Patterns to Avoid

### ❌ Don't Do This
```bash
# Massive commit with multiple unrelated changes
git commit -m "feat: implement entire API layer with all services"

# Vague commit messages
git commit -m "fix stuff"

# Committing work in progress
git commit -m "WIP: working on API"

# Mixing different types of changes
git commit -m "feat: add API and fix database bug"
```

### ✅ Do This Instead
```bash
# Atomic commits for each sub-task
git commit -m "feat(api-gateway): implement REST API endpoints"
git commit -m "feat(query-service): add Redis caching layer"
git commit -m "fix(database): correct connection pool configuration"

# Clear, descriptive messages
git commit -m "feat(api-gateway): add event filtering with pagination support"

# Complete, tested changes only
git commit -m "feat(query-service): implement SQL query optimizer with tests"
```

## Integration with Development Phases

### Phase Planning
1. **Break down each phase into sub-tasks**
2. **Estimate time for each sub-task**
3. **Plan commits for each sub-task**
4. **Track progress with commit history**

### Phase Completion
1. **All sub-tasks committed**
2. **All tests passing**
3. **Documentation updated**
4. **Ready for code review**
5. **Merge to main branch**

## Tools and Automation

### 1. Pre-commit Hooks
```bash
# Install pre-commit hooks
make install-hooks

# Hooks will run:
# - go fmt
# - go vet
# - golangci-lint
# - commit message format check
```

### 2. Commit Message Templates
```bash
# Set up commit message template
git config commit.template .gitmessage

# Template content:
# type(scope): brief description
#
# Detailed description:
# - What was changed
# - Why it was changed
# - Any breaking changes
#
# Resolves: Phase X Task Y - Task Description
```

### 3. Automated Checks
```bash
# Check commit message format
make check-commit-msg

# Validate commit history
make check-commit-history

# Generate changelog from commits
make generate-changelog
```

## Examples by Phase

### Phase 1: Infrastructure
```bash
feat(infrastructure): setup mono-repo with Go workspace
feat(docker): add development docker-compose configuration
feat(ci): configure GitHub Actions for automated testing
chore(deps): initialize Go modules for all services
```

### Phase 2: Indexer Service
```bash
feat(indexer): implement blockchain event monitoring
feat(indexer): add event parsing with go-ethereum
feat(indexer): implement reorg handling logic
feat(database): add PostgreSQL schema and migrations
```

### Phase 3: API Layer
```bash
feat(graphql): design complete GraphQL schema
feat(grpc): define service interfaces
feat(query-service): implement gRPC server with caching
feat(api-gateway): add REST API endpoints
feat(admin-service): implement contract management
feat(testing): add integration tests and documentation
```

## Conclusion

This git workflow ensures:
- **Clean, reviewable commit history**
- **Easy debugging and rollback**
- **Clear progress tracking**
- **Better code review process**
- **Maintainable project structure**

Follow these guidelines to maintain high code quality and enable effective collaboration throughout the project development.
