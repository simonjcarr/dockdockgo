# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

A docker swarm like application that has the following features

- Written in GoLang
- A CLI with mirror API
- CLI can generate API tokens for secure API access
- The API will be used for external applications and GUI development

## User experience
The key to this application will be simplicity. It should be easy to simply install, specify an image, or docker-compose file, a list of remote servers and immediatley have a cluster of containers running in a highly available fashion. As much complexity and settings should be hidden from the user by using sensible defaults, but the user should be able to override important settings when required.

The API will set it appart from docker and docker swarm in order to allow easy remote management from other applications and web interfaces to be developed.

## Key Project Features
- Search remote registries for public and private images, if remote is not specified it should default to DockerHub
- Search for images locally
- User should be able to start a container from a local or remote image and specify all the settings they would by using the docker command
- The application should be able to read standard docker-compose files so it is backward compatible with docker compose
- When running a single container, the user should be able to provide a single remote server or list of remote servers to run the container on.
- The application should support an extended docker-compose file structure, where a list of remote servers can be provided for all or some of the containers to run on.
- The application should support the ability to provide an ssh key or username and password to connect to remote servers via SSH
- When connecting to a remote server, the app should check if docker is installed, if not it should prompt the user to confirm installing docker, but should also accept a switch to the dockdockgo command to install docker if not installed without asking.
- There will be option to specify how many replicas of the container should be run
- Where a container has port mappings specified, the app should ensure that their are no conflicts if the same container is run multiple times on the same server
- The app will install it's self on the remote machines along with zoo keeper. One instance of the app will be the master, and if not available another instance will take over as master.
- The each instance of the app on each server, will act as a tcp and http router supporting both http and https and will route traffic based on different policies (configurable) in order to share the traffic load between container instances. It should run an instance of redis in docker on each server in order to take care of managing instance availability, port mappings, and routing settings etc.
- The app should support using lets encrypt and self signed certificates for https

## Development Workflow & Automation

This project uses **fully automated semantic versioning** and **professional CI/CD workflows**. Always follow these guidelines:

### Branch Strategy
- **main branch**: Production releases (`1.0.0`, `1.2.0`, `2.0.0`)
- **develop branch**: Development releases (`1.0.0-beta.1`, `1.1.0-beta.2`)
- **feature branches**: Individual features/fixes (`feat/new-feature`, `fix/bug-name`)

### Feature Branch Workflow
**ALWAYS create feature branches for ANY changes - never commit directly to develop or main:**

1. **Create focused feature branches** for small, related chunks of work:
   ```bash
   git checkout -b feat/descriptive-feature-name  # for new features
   git checkout -b fix/descriptive-fix-name       # for bug fixes
   git checkout -b docs/descriptive-doc-name      # for documentation
   ```

2. **Keep feature branches small and focused** - one logical change per branch:
   - ✅ `feat/add-gRPC-server` - implements gRPC communication
   - ✅ `fix/container-hanging-issue` - fixes specific container bug
   - ✅ `docs/update-api-documentation` - updates API docs
   - ❌ `feat/massive-refactor-everything` - too broad

3. **Use conventional commit messages** (this drives automatic versioning):
   ```
   feat: add new feature that enables X
   fix: resolve issue with Y component  
   docs: update API documentation
   style: format code with gofmt
   refactor: restructure database layer
   test: add integration tests for auth
   build: update CI/CD pipeline
   chore: update dependencies
   ```

### Pull Request Process
**ALWAYS merge feature branches via pull requests:**

1. **Create pull request** with descriptive title using conventional format:
   ```
   feat: implement gRPC cluster communication
   fix: resolve container hanging on remote servers
   docs: add API authentication examples
   ```

2. **Pull requests are automatically merged** - no manual intervention needed
3. **CI/CD runs automatically** on every pull request:
   - Code formatting checks (`gofmt`)
   - Tests execution (`go test ./...`)
   - Build verification
   - Semantic versioning

### Automated Versioning System
**Semantic versioning is fully automated** based on conventional commits:

- **feat:** commits → Minor version bump (`1.0.0` → `1.1.0`)
- **fix:** commits → Patch version bump (`1.0.0` → `1.0.1`)
- **BREAKING CHANGE:** → Major version bump (`1.0.0` → `2.0.0`)

**Release Examples:**
- **Develop branch**: `v1.0.0-beta.1`, `v1.1.0-beta.2` (pre-releases)
- **Main branch**: `v1.0.0`, `v1.1.0`, `v2.0.0` (production releases)

### Work Chunking Guidelines
**Break work into small, logical feature branches:**

✅ **Good Examples:**
- `feat/add-health-check-endpoint` - Single API endpoint
- `fix/port-conflict-resolution` - Specific bug fix
- `feat/implement-container-scheduling` - One component
- `docs/deployment-guide` - Focused documentation

❌ **Avoid These:**
- `feat/complete-distributed-system` - Too broad
- `fix/various-bugs` - Multiple unrelated fixes
- `refactor/everything` - Too sweeping

### Progress Tracking
- **Use TodoWrite tool** extensively to track progress and plan work
- **Update PLAN.md** when discovering new features to implement
- **Mark completed items** in PLAN.md as work is finished
- **Always complete merges** of feature branches into develop

### Version Consistency
- **Local builds** use same versioning as GitHub releases
- **Build artifacts** include proper semantic versions
- **Git tags** drive version detection automatically

### Key Commands
```bash
# Create and work on feature branch
git checkout -b feat/your-feature
# ... make changes ...
git add .
git commit -m "feat: add your feature description"
git push --set-upstream origin feat/your-feature

# Create and auto-merge pull request  
gh pr create --title "feat: add your feature" --body "Description" --base develop
gh pr merge --squash --delete-branch
```

**Remember: Every change goes through feature branch → PR → automated merge → automated release**

## Documentation
You create a README.md file that contains instructions on how to use dockdockgo
The documentation should be simple and concise.
You should also create a folder with html pages of instructions and examples in the use of dockdockgo. There should be an index.html file that provides a getting started section and a links in logical order to the other html pages. This will eventually be used to create a support web site for the dockdockgo application.

