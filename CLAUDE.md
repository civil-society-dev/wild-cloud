# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This project is called "Wild Cloud". Wild Cloud is a platform for managing and orchestrating cloud-native applications on local networks using a network appliance called "Wild Central".

Wild Central is a lightweight server that runs on a local machine (e.g., a Raspberry Pi) and provides an API for users to manage their Wild Cloud instances. The Wild Cloud API is implemented in the api project. @api/CLAUDE.md . Wild Central devices can be set up using our apt package implemented in the dist project. @dist/README.md

All data and config for an instance of Wild Central is defined in Central's environment in the WILD_API_DATA_DIR variable. For ease of access, a symlink to our Wild API data dir is made at ./wild-cloud-redmond-data.

A Wild Cloud instance is a kubernetes (k8s) environment that runs Wild Cloud services and applications. Wild Cloud instances can be created, managed, and monitored using the Wild Cloud API running on a Wild Central device.

Wild Cloud applications are custom packages designed to be deployed to Wild Cloud instances. They consist of kustomize templates and a Wild Cloud app manifest file that describes the application and how it should be deployed configured and deployed in a Wild Cloud instance. Wild Cloud applications are stored in a "Wild Directory". The directory contained in the wild-directory folder is the official Wild Directory. @wild-directory/CLAUDE.md

The Wild Cloud API maintains data for each Wild Cloud instance in its configured WILD_API_DATA_DIR. A data directory is intended to be checked into version control (e.g., git) to track changes to the configuration of Wild Cloud instances and their deployed applications over time. These are designed to follow infrastructure-as-code principles, allowing experienced devops users to manage their Wild Cloud instances using familiar tools and workflows. 

We provide a command-line interface (CLI) tool called Wild CLI, implemented in the cli project, that allows users to interact with the Wild Cloud API and manage their Wild Cloud instances from the terminal. This allows users to automate tasks and integrate Wild Cloud management into their existing workflows. @cli/README.md

To make Wild Cloud more accessible to less-experienced users, the Wild Central device hosts a web-based interface for managing Wild Cloud instances, which is implemented in the web project. @web/CLAUDE.md

## Additional Documentation

### Info about Talos

- @ai/talos-v1.11/README.md
- @ai/talos-v1.11/architecture-and-components.md
- @ai/talos-v1.11/cli-essentials.md
- @ai/talos-v1.11/cluster-operations.md
- @ai/talos-v1.11/discovery-and-networking.md
- @ai/talos-v1.11/etcd-management.md
- @ai/talos-v1.11/bare-metal-administration.md
- @ai/talos-v1.11/troubleshooting-guide.md

## Implementation Philosophy

## Core Philosophy

Embodies a Zen-like minimalism that values simplicity and clarity above all. This approach reflects:

- **Wabi-sabi philosophy**: Embracing simplicity and the essential. Each line serves a clear purpose without unnecessary embellishment.
- **KISS**: The solution should be as simple as possible, but no simpler.
- **YAGNI**: Avoid building features or abstractions that aren't immediately needed. The code handles what's needed now rather than anticipating every possible future scenario.
- **Trust in emergence**: Complex systems work best when built from simple, well-defined components that do one thing well.
- **Pragmatic trust**: The developer trusts external systems enough to interact with them directly, handling failures as they occur rather than assuming they'll happen.
- **Consistency is key**: Uniform patterns and conventions make the codebase easier to understand and maintain. If you introduce a new pattern, make sure it's consistently applied. There should be one obvious way to do things.

This development philosophy values clear, concise documentation, readable code, and belief that good architecture emerges from simplicity rather than being imposed through complexity.

## Core Design Principles

### 1. Ruthless Simplicity

- **KISS principle taken to heart**: Keep everything as simple as possible, but no simpler
- **Minimize abstractions**: Every layer of abstraction must justify its existence
- **Start minimal, grow as needed**: Begin with the simplest implementation that meets current needs
- **Avoid future-proofing**: Don't build for hypothetical future requirements
- **Question everything**: Regularly challenge complexity in the codebase

### 2. Architectural Integrity with Minimal Implementation

- **Preserve key architectural patterns**: Maintain clear boundaries and responsibilities
- **Simplify implementations**: Maintain pattern benefits with dramatically simpler code
- **Scrappy but structured**: Lightweight implementations of solid architectural foundations
- **End-to-end thinking**: Focus on complete flows rather than perfect components

### 3. Library vs Custom Code

Choosing between custom code and external libraries is a judgment call that evolves with your requirements. There's no rigid rule - it's about understanding trade-offs and being willing to revisit decisions as needs change.

#### The Evolution Pattern

Your approach might naturally evolve:
- **Start simple**: Custom code for basic needs (20 lines handles it)
- **Growing complexity**: Switch to a library when requirements expand
- **Hitting limits**: Back to custom when you outgrow the library's capabilities

This isn't failure - it's natural evolution. Each stage was the right choice at that time.

#### When Custom Code Makes Sense

Custom code often wins when:
- The need is simple and well-understood
- You want code perfectly tuned to your exact requirements
- Libraries would require significant "hacking" or workarounds
- The problem is unique to your domain
- You need full control over the implementation

#### When Libraries Make Sense

Libraries shine when:
- They solve complex problems you'd rather not tackle (auth, crypto, video encoding)
- They align well with your needs without major modifications
- The problem is well-solved with mature, battle-tested solutions
- Configuration alone can adapt them to your requirements
- The complexity they handle far exceeds the integration cost

#### Making the Judgment Call

Ask yourself:
- How well does this library align with our actual needs?
- Are we fighting the library or working with it?
- Is the integration clean or does it require workarounds?
- Will our future requirements likely stay within this library's capabilities?
- Is the problem complex enough to justify the dependency?

#### Recognizing Misalignment

Watch for signs you're fighting your current approach:
- Spending more time working around the library than using it
- Your simple custom solution has grown complex and fragile  
- You're monkey-patching or heavily wrapping a library
- The library's assumptions fundamentally conflict with your needs

#### Stay Flexible

Remember that complexity isn't destroyed, only moved. Libraries shift complexity from your code to someone else's - that's often a great trade, but recognize what you're doing.

The key is avoiding lock-in. Keep library integration points minimal and isolated so you can switch approaches when needed. There's no shame in moving from custom to library or library to custom. Requirements change, understanding deepens, and the right answer today might not be the right answer tomorrow. Make the best decision with current information, and be ready to evolve.

## Technical Implementation Guidelines

### API Layer

- Implement only essential endpoints
- Minimal middleware with focused validation
- Clear error responses with useful messages
- Consistent patterns across endpoints

### Storage

- Prefer simple file storage
- Simple schema focused on current needs

## Development Approach

### Vertical Slices

- Implement complete end-to-end functionality slices
- Start with core user journeys
- Get data flowing through all layers early
- Add features horizontally only after core flows work

### Iterative Implementation

- 80/20 principle: Focus on high-value, low-effort features first
- One working feature > multiple partial features
- Validate with real usage before enhancing
- Be willing to refactor early work as patterns emerge

### Testing Strategy

- Focus on critical path testing initially
- Add unit tests for complex logic and edge cases
- Testing pyramid: 60% unit, 30% integration, 10% end-to-end

### Error Handling

- Handle common errors robustly
- Log detailed information for debugging
- Provide clear error messages to users
- Fail fast and visibly during development

## Decision-Making Framework

When faced with implementation decisions, ask these questions:

1. **Necessity**: "Do we actually need this right now?"
2. **Simplicity**: "What's the simplest way to solve this problem?"
3. **Directness**: "Can we solve this more directly?"
4. **Value**: "Does the complexity add proportional value?"
5. **Maintenance**: "How easy will this be to understand and change later?"

## Areas to Embrace Complexity

Some areas justify additional complexity:

1. **Security**: Never compromise on security fundamentals
2. **Data integrity**: Ensure data consistency and reliability
3. **Core user experience**: Make the primary user flows smooth and reliable
4. **Error visibility**: Make problems obvious and diagnosable

## Areas to Aggressively Simplify

Push for extreme simplicity in these areas:

1. **Internal abstractions**: Minimize layers between components
2. **Generic "future-proof" code**: Resist solving non-existent problems
3. **Edge case handling**: Handle the common cases well first
4. **Framework usage**: Use only what you need from frameworks
5. **State management**: Keep state simple and explicit

## Remember

- It's easier to add complexity later than to remove it
- Code you don't write has no bugs
- Favor clarity over cleverness
- The best code is often the simplest

This philosophy document serves as the foundational guide for all implementation decisions in the project.

