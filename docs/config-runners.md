# Runner Configuration

The MCP CLI Adapter supports multiple execution runners that allow tools to run in different environments with various security restrictions. This document details how to configure and use these runners.

For basic configuration information, see [Configuration Overview](config.md).

## Multiple Runners and Selection Process

You can define multiple runners for a tool to support different execution environments. The system will select the first runner whose requirements are satisfied by the current system.

Each runner definition includes:

- `name`: The name of the runner (e.g., "sandbox-exec", "firejail", "exec")
- `requirements`: System requirements that must be met for this runner to be available
  - `os`: Operating system name (e.g., "darwin", "linux", "windows")
  - `executables`: List of executables that must be present in the system PATH
- `options`: Configuration options specific to the runner

Here's an example of a tool with multiple runners:

```yaml
run:
  command: "echo 'Hello {{ .name }}'"
  runners:
    - name: sandbox-exec
      requirements:
        os: darwin
        executables: [sandbox-exec]
      options:
        allow_networking: false
        allow_user_folders: false
    - name: firejail
      requirements:
        os: linux
        executables: [firejail]
      options:
        allow_networking: false
        allow_user_folders: false
    - name: exec
      requirements: {}  # No specific requirements, acts as a fallback
```

In this example:

1. On macOS with sandbox-exec available, the sandbox-exec runner will be used
2. On Linux with firejail available, the firejail runner will be used
3. On any other system, the exec runner will be used as a fallback

**Important Notes on Runner Selection:**

- The `runners` array is **optional**. If not provided, a default "exec" runner with no sandboxing will be used.
- If you do specify `runners`, at least one of them must meet its requirements for the tool to be available.
- No automatic fallback to "exec" occurs if you specify `runners` but none meet their requirements.
- If you want a fallback, explicitly add an `exec` runner with empty requirements at the end of your runners list.

It's recommended to always include a fallback runner (typically named "exec" with no requirements) to ensure your tool can run on any platform if you want it to be universally available.

## Runner Types

### Default Runner (exec)

The default runner executes commands directly on the host system using the configured shell. It has no special requirements or sandboxing.

```yaml
runners:
  - name: exec
    requirements: {}  # No specific requirements
```

### Sandbox Runner (macOS Only)

The sandbox runner uses macOS's `sandbox-exec` command to run commands in a sandboxed environment with restricted access to the system. This provides an additional layer of security by restricting what commands can access.

```yaml
runners:
  - name: sandbox-exec
    requirements:
      os: darwin
      executables: [sandbox-exec]
    options:
      allow_networking: false           # Disable network access
      allow_user_folders: false         # Restrict access to user folders
      allow_read_folders:               # List of folders to explicitly allow access to
        - "/tmp"
        - "/path/to/project"
```

#### Sandbox Configuration Options

Available options:

- `allow_networking`: When set to `false`, blocks all network access
- `allow_user_folders`: When set to `false`, restricts access to user folders like Documents, Desktop, etc.
- `allow_read_folders`: List of directories to explicitly allow access to read, even when other
  restrictions are in place. Items in this list can use Golang template replacements (using the tool parameters).
- `allow_write_folders`: List of directories to explicitly allow access to write, even when other
  restrictions are in place. Items in this list can use Golang template replacements (using the tool parameters).
- `custom_profile`: Specify a custom sandbox profile for advanced configuration

#### Custom Sandbox Profiles

For advanced usage, you can specify a completely custom sandbox profile using the `custom_profile` option.

Here's an example of a custom profile that:

- Allows most operations by default
- Denies network access
- Allows read access only to /tmp and system directories

```yaml
runners:
  - name: sandbox-exec
    requirements:
      os: darwin
      executables: [sandbox-exec]
    options:
      custom_profile: |
        (version 1)
        (allow default)
        (deny network*)
        (allow file-read-data (regex "^/tmp"))
```

### Firejail Runner (Linux Only)

The firejail runner uses [firejail](https://firejail.wordpress.com/) to run commands in a sandboxed environment on Linux systems. Firejail is a SUID sandbox program that restricts the running environment of untrusted applications using Linux namespaces and seccomp-bpf.

```yaml
runners:
  - name: firejail
    requirements:
      os: linux
      executables: [firejail]
    options:
      allow_networking: false           # Disable network access
      allow_user_folders: false         # Restrict access to user folders
      allow_read_folders:               # List of folders to explicitly allow access to
        - "/tmp"
        - "/etc/ssl/certs"
```

#### Requirements

- Linux operating system
- Firejail installed (`apt-get install firejail` on Debian/Ubuntu or equivalent for your distribution)

#### Firejail Configuration Options

Available options:

- `allow_networking`: When set to `false`, blocks all network access using `net none`
- `allow_user_folders`: When set to `false`, restricts access to common user folders like Documents, Desktop, etc.
- `allow_read_folders`: List of directories to explicitly allow read access to, even when other restrictions
  are in place. Items in this list can use Golang template replacements (using the tool parameters).
- `allow_write_folders`: List of directories to explicitly allow both read and write access to.
  Items in this list can use Golang template replacements (using the tool parameters).
- `custom_profile`: Specify a custom firejail profile for advanced configuration

#### Security Benefits

The firejail runner adds several layers of security:

1. **Filesystem isolation**: Restricts access to sensitive directories
2. **Network restrictions**: Can completely disable network access
3. **System call filtering**: Uses seccomp-bpf to restrict available system calls
4. **Capabilities restrictions**: Drops dangerous capabilities
5. **No root access**: Prevents elevation to root privileges

#### Custom Firejail Profiles

For advanced usage, you can specify a completely custom firejail profile using the `custom_profile` option:

```yaml
runners:
  - name: firejail
    requirements:
      os: linux
      executables: [firejail]
    options:
      custom_profile: |
        # Custom firejail profile
        net none
        blacklist ${HOME}
        seccomp
        caps.drop all
        noroot
```

### Docker Runner

The Docker runner executes commands inside Docker containers, providing strong isolation from the host system. This runner creates a temporary script file containing your command, then mounts it into a Docker container and executes it.

```yaml
runners:
  - name: docker
    requirements:
      executables: [docker]
    options:
      image: "alpine:latest"            # Required: Docker image to use
      allow_networking: true            # Optional: Allow network access (default: true)
      mounts:                           # Optional: Additional volumes to mount
        - "/data:/data:ro"              # Format: "host-path:container-path[:options]"
        - "/config:/etc/myapp:ro"
      user: "1000:1000"                 # Optional: User to run as in container
      workdir: "/app"                   # Optional: Working directory in container
      docker_run_opts: "--cpus 1 --memory 512m"  # Optional: Additional docker run options
      prepare_command: |
        # Commands to run before the main command
        apt-get update
        apt-get install -y python3
```

#### Requirements

- Docker installed and available in PATH
- Appropriate permissions to run Docker containers (typically membership in the `docker` group or root)

#### Docker Configuration Options

Available options:

- `image`: (Required) The Docker image to use for running the command (e.g., "alpine:latest", "ubuntu:22.04")
- `allow_networking`: When set to `false`, disables all network access for the container using `--network none`
- `mounts`: A list of additional volumes to mount in the format "host-path:container-path[:options]"
- `user`: Specify the user to run as within the container (format: "uid" or "uid:gid")
- `workdir`: Set the working directory inside the container
- `docker_run_opts`: String of additional options to pass to the `docker run` command
- `prepare_command`: Commands to run before the main command (e.g., for installing packages or setting up the environment)

#### Security Benefits

The Docker runner provides several security advantages:

1. **Complete process isolation**: Processes inside the container are isolated from the host
2. **Configurable resource limits**: Can limit CPU, memory, and other resources
3. **Control over capabilities**: Docker restricts Linux capabilities by default
4. **Filesystem isolation**: Only mounted volumes are accessible
5. **Network isolation**: Can completely disable network access
6. **User namespace separation**: Can run as non-root inside the container

#### Examples

##### Basic Alpine Container

```yaml
runners:
  - name: docker
    requirements:
      executables: [docker]
    options:
      image: "alpine:latest"
```

##### Limited Resources Python Container

```yaml
runners:
  - name: docker
    requirements:
      executables: [docker]
    options:
      image: "python:3.9-slim"
      docker_run_opts: "--cpus 0.5 --memory 256m --read-only"
      allow_networking: false
      workdir: "/app"
      user: "nobody"
```

##### Data Analysis Container With Volume Mounts

```yaml
runners:
  - name: docker
    requirements:
      executables: [docker]
    options:
      image: "jupyter/datascience-notebook:latest"
      mounts:
        - "{{ .datadir }}:/data:ro"
        - "/tmp:/tmp"
      workdir: "/data"
```

##### Container With Package Installation

```yaml
runners:
  - name: docker
    requirements:
      executables: [docker]
    options:
      image: "debian:bullseye-slim"
      prepare_command: |
        # Update package lists
        apt-get update -y
        
        # Install required packages
        apt-get install -y --no-install-recommends \
          curl \
          jq \
          ca-certificates
        
        # Clean up to reduce container size
        apt-get clean
        rm -rf /var/lib/apt/lists/*
      allow_networking: true
```

The `prepare_command` is executed before the main command and can be used to install dependencies, configure the environment, or perform any setup tasks needed for the command to run successfully. This is especially useful for lightweight base images where you need to install additional tools.

## Cross-Platform Example

Here's a complete example of a tool that uses different runners based on the platform:

```yaml
- name: "safe_file_read"
  description: "Reads a file safely using appropriate sandboxing for the platform"
  params:
    filename:
      type: string
      description: "Path to the file to read"
      required: true
  constraints:
    - "filename.size() > 0"                               # Filename must not be empty
    - "!filename.contains('../')"                         # Prevent directory traversal
    - "['.txt', '.log', '.md'].exists(ext, filename.endsWith(ext))"  # Only allow certain file extensions
  run:
    command: "cat {{ .filename }}"
    runners:
      - name: sandbox-exec
        requirements:
          os: darwin
          executables: [sandbox-exec]
        options:
          allow_networking: false
          allow_user_folders: false
          allow_read_folders:
            - "/tmp"
            - "{{ .filename }}"
      - name: firejail
        requirements:
          os: linux
          executables: [firejail]
        options:
          allow_networking: false
          allow_user_folders: false
          allow_read_folders:
            - "/tmp"
            - "{{ .filename }}"
      - name: exec
        requirements: {}  # Fallback runner for other platforms
  output:
    prefix: "Contents of {{ .filename }}:"
``` 