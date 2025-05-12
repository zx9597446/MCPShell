# Configuration File

The MCP CLI Adapter can be configured using a YAML configuration file to define the tools that the MCP server provides.

## Basic Structure

The configuration file uses the following structure:

```yaml
mcp:
  run:
    shell: "<shell>"
  description: <global description>
  tools:
    - name: "<tool_name>"
      description: "<tool description>"
      params:
        param_name:
          type: <string|number|boolean>
          description: "<parameter description>"
          required: <true|false>
      constraints:
        - "<constraint expression>"
      run:
        command: "<command to execute>"
      output:
        prefix: "<text to prepend to the output>"
```

## MCP-CLI Configuration

The top-level `mcp` section contains configuration for the MCP server:

- `description`: gglobal description of the toolkit.
- `run`: Run configuration settings
  - `shell`: Optional string specifying which shell to use for command execution.
    If not provided, the system will use the SHELL environment variable or fall back to `/bin/sh`.
- `tools`: Array of tool definitions (required)

## Tools Definitions

Each tool is defined with the following properties:

- `name`: The name of the tool (required)
- `description`: A description of what the tool does (required).
  This is specially important in order to instruct the LLM what this tool does.
  Otherwise, the LLM will not know that it can use this tool for fullfilling
  the user requests.
- `params`: A map of parameters that the tool accepts
- `constraints`: A list of CEL expressions to validate before command execution (optional)
- `run`: Configuration for how the tool executes (required)
- `output`: Configuration for tool output formatting (optional)

### Parameter Definition

Each parameter has the following properties:

- `type`: The parameter type (string, number, or boolean). Optional, defaults to "string" if not specified.
- `description`: A description of the parameter
- `required`: Whether the parameter is required (default: false)

### Constraints

Constraints are optional [CEL (Common Expression Language)](https://github.com/google/cel-spec)
expressions that are evaluated before command execution.
The tool will only execute if all constraints evaluate to `true`.
This provides a safety mechanism to prevent
potentially dangerous commands from being executed.

Constraints have access to all tool parameters by name. For example:

```yaml
constraints:
  - "text.startsWith('text:')"  # Ensures the text parameter starts with "text:"
  - "command.size() < 100"      # Ensures the command parameter is less than 100 characters
```

#### Understanding CEL Constraint Language

CEL (Common Expression Language) is a simple, portable expression language developed by Google. In the MCP CLI Adapter, CEL is used to define safety constraints that validate parameters before command execution.

##### Basic Types and Operations

CEL supports three basic parameter types in MCP CLI Adapter:

1. **String operations**:

   - `string.size()` - Returns the length of the string
   - `string.contains(substring)` - Checks if a string contains a substring
   - `string.startsWith(prefix)` - Checks if a string starts with a prefix
   - `string.endsWith(suffix)` - Checks if a string ends with a suffix
   - `string.matches(regex)` - Checks if a string matches a regular expression

   ```yaml
   constraints:
     - "name.size() <= 50"                      # Limit string length
     - "!filename.contains('../')"              # Prevent directory traversal
     - "text.matches('^[a-zA-Z0-9 ,.!?]*$')"    # Only allow alphanumeric and basic punctuation
   ```

2. **Numeric operations**:

   - Comparison operators: `==`, `!=`, `<`, `<=`, `>`, `>=`
   - Arithmetic operators: `+`, `-`, `*`, `/`, `%`

   ```yaml
   constraints:
     - "value > 0.0"                # Ensure positive values
     - "value <= 100.0"             # Set upper limit
     - "count % 2 == 0"             # Ensure even numbers only
   ```

3. **Boolean operations**:
   - Logical operators: `&&` (and), `||` (or), `!` (not)

   ```yaml
   constraints:
     - "flag == true"               # Check if flag is true
     - "!dangerous"                 # Ensure dangerous flag is not set
     - "valid && authorized"        # Require both valid and authorized flags
   ```

##### Advanced Features

1. **List operations and quantifiers**:

   - `list.exists(var, condition)` - Checks if any element in the list satisfies the condition
   - `list.all(var, condition)` - Checks if all elements in the list satisfy the condition

   ```yaml
   constraints:
     - "['ls', 'echo', 'cat'].exists(cmd, cmd == command)"  # Whitelist allowed commands
     - "['.jpg', '.png', '.gif'].exists(ext, filename.endsWith(ext))"  # Only allow certain file extensions
   ```

2. **Combining multiple constraints**:
   - Multiple constraints are implicitly AND-ed together
   - Use `||` inside a single constraint expression for OR logic

   ```yaml
   constraints:
     - "command == 'ls' || command == 'pwd'"  # Allow only ls or pwd
     - "!command.contains('rm')"              # Never allow rm command
   ```

##### Common Constraint Patterns

1. **Security constraints** to prevent command injection:

   ```yaml
   constraints:
     - "!text.contains(';')"                    # Prevent command chaining
     - "!text.contains('&')"                    # Prevent background execution
     - "!text.contains('|')"                    # Prevent piping
     - "!text.matches('.*[;&|`$].*')"           # Block shell special characters
   ```

2. **Filesystem safety constraints**:

   ```yaml
   constraints:
     - "!path.contains('../')"                  # Prevent directory traversal
     - "path.startsWith('/allowed/dir/')"       # Only allow specific directory
     - "path.matches('^[a-zA-Z0-9_\\-./]+$')"   # Only allow safe path characters
   ```

3. **Command whitelisting**:

   ```yaml
   constraints:
     - "['ls', 'ps', 'echo', 'cat', 'grep'].exists(c, c == command)"  # Allow only specific commands
   ```

4. **Input validation constraints**:

   ```yaml
   constraints:
     - "email.matches('^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$')"  # Validate email format
     - "phone.matches('^\\+?[0-9]{10,15}$')"                                 # Validate phone number
   ```

##### Complete Examples

Here's a more comprehensive example demonstrating various constraints:

```yaml
mcp:
  tools:
    - name: "secure_file_reader"
      description: "Safely read file contents"
      params:
        filename:
          type: string
          description: "Path to the file to read"
          required: true
      constraints:
        - "filename.size() > 0"                                # Filename must not be empty
        - "filename.size() < 255"                              # Filename must be reasonable length
        - "!filename.contains('../')"                          # Prevent directory traversal
        - "!filename.startsWith('/')"                          # Prevent absolute paths
        - "['.txt', '.log', '.md', '.json'].exists(ext, filename.endsWith(ext))"  # Only allow certain file extensions
      run:
        command: "cat {{ .filename }}"

    - name: "restricted_shell"
      description: "Run only safe commands"
      params:
        command:
          type: string
          description: "Command to run"
          required: true
        arguments:
          type: string
          description: "Command arguments"
          required: false
      constraints:
        - "['ls', 'pwd', 'echo', 'cat', 'grep', 'find'].exists(cmd, cmd == command)"  # Whitelist allowed commands
        - "!arguments.contains(';')"                          # Prevent command chaining
        - "!arguments.contains('&')"                          # Prevent background execution
        - "!arguments.contains('|')"                          # Prevent piping
        - "!arguments.contains('>')"                          # Prevent redirection
        - "arguments.size() < 100"                            # Limit argument length
      run:
        command: "{{ .command }} {{ .arguments }}"
```

For more information on CEL syntax, see the [CEL GitHub repository](https://github.com/google/cel-go).

###  `run` Configuration

The run configuration defines how the tool executes:

- `command`: A shell command to execute (required)
- `env`: A list of environment variable names to pass from the parent process to the command (optional)
- `runner`: The type of runner to use for executing the command (optional, defaults to "exec")
- `options`: Configuration options specific to the runner being used (optional)

Commands can include parameter values using Go template syntax with `{{ .param_name }}`.

Example specifying environment variables to pass through:

```yaml
run:
  env:
    - KUBECONFIG     # Pass the KUBECONFIG environment variable to the command
    - HOME           # Pass the HOME environment variable to the command
  command: |
    kubectl get {{ .resource }}
```

This is useful for tools that need access to environment variables like API keys, configuration paths, or user information.

#### Runner Types

##### Default Runner

The default runner executes commands directly on the host system using the configured shell.

##### Sandbox Runner (macOS Only)

The sandbox runner uses macOS's `sandbox-exec` command to run commands in a sandboxed environment with restricted access to the system. This provides an additional layer of security by restricting what commands can access.

To use the sandbox runner, specify `runner: sandbox-exec` in your tool's `run` configuration block:

```yaml
run:
  runner: sandbox-exec  # Specify the sandbox runner here
  command: 'echo "Hello {{.param1}}"'
```

By default, the sandbox runner uses a permissive profile that allows most operations but can be restricted as needed.

###### Sandbox Configuration Options

You can customize the sandbox behavior using the `options` field in your `run` configuration:

```yaml
run:
  runner: sandbox-exec
  command: '{{.command}}'
  options:
    allow_networking: false           # Disable network access
    allow_user_folders: false         # Restrict access to user folders
    allow_read_folders:               # List of folders to explicitly allow access to
      - "/tmp"
      - "/path/to/project"
      - {{ .param1 }}                 # Go templates can be used
```

Available options:

- `allow_networking`: When set to `false`, blocks all network access
- `allow_user_folders`: When set to `false`, restricts access to user folders like Documents, Desktop, etc.
- `allow_read_folders`: List of directories to explicitly allow access to read, even when other
  restrictions are in place. Items in this list can use Golang template replacements (using the tool parameters).
- `allow_write_folders`: List of directories to explicitly allow access to write, even when other
  restrictions are in place. Items in this list can use Golang template replacements (using the tool parameters).
- `custom_profile`: Specify a custom sandbox profile for advanced configuration

###### Custom Sandbox Profiles

For advanced usage, you can specify a completely custom sandbox profile using the `custom_profile` option.

Here's an example of a custom profile that:

- Allows most operations by default
- Denies network access
- Allows read access only to /tmp and system directories

```yaml
run:
  runner: sandbox-exec
  options:
    custom_profile: |
      (version 1)
      (allow default)
      (deny network*)
      (allow file-read-data (regex "^/tmp"))
```

See the [examples/sandbox-config.yaml](../examples/sandbox-config.yaml) file for complete examples.

##### Firejail Runner (Linux Only)

The firejail runner uses [firejail](https://firejail.wordpress.com/) to run commands in a sandboxed environment on Linux systems. Firejail is a SUID sandbox program that restricts the running environment of untrusted applications using Linux namespaces and seccomp-bpf.

To use the firejail runner, specify `runner: firejail` in your tool's `run` configuration block:

```yaml
run:
  runner: firejail  # Specify the firejail runner here
  command: 'echo "Hello {{.param1}}"'
```

###### Requirements

- Linux operating system
- Firejail installed (`apt-get install firejail` on Debian/Ubuntu or equivalent for your distribution)

###### Firejail Configuration Options

You can customize the firejail behavior using the `options` field in your `run` configuration:

```yaml
run:
  runner: firejail
  command: '{{.command}}'
  options:
    allow_networking: false           # Disable network access
    allow_user_folders: false         # Restrict access to user folders
    allow_read_folders:               # List of folders to explicitly allow read access to
      - "/tmp"
      - "/etc/ssl/certs"
    allow_write_folders:              # List of folders to explicitly allow write access to
      - "/tmp/downloads"
```

Available options:

- `allow_networking`: When set to `false`, blocks all network access using `net none`
- `allow_user_folders`: When set to `false`, restricts access to common user folders like Documents, Desktop, etc.
- `allow_read_folders`: List of directories to explicitly allow read access to, even when other restrictions
  are in place. Items in this list can use Golang template replacements (using the tool parameters).
- `allow_write_folders`: List of directories to explicitly allow both read and write access to.
  Items in this list can use Golang template replacements (using the tool parameters).
- `custom_profile`: Specify a custom firejail profile for advanced configuration

###### Security Benefits

The firejail runner adds several layers of security:

1. **Filesystem isolation**: Restricts access to sensitive directories
2. **Network restrictions**: Can completely disable network access
3. **System call filtering**: Uses seccomp-bpf to restrict available system calls
4. **Capabilities restrictions**: Drops dangerous capabilities
5. **No root access**: Prevents elevation to root privileges

###### Custom Firejail Profiles

For advanced usage, you can specify a completely custom firejail profile using the `custom_profile` option:

```yaml
run:
  runner: firejail
  options:
    custom_profile: |
      # Custom firejail profile
      net none
      blacklist ${HOME}
      seccomp
      caps.drop all
      noroot
```

### `output` Configuration

The output configuration defines how the tool's output is formatted:

- `prefix`: Text to prepend to the command output (optional)

Similar to commands, prefixes can include parameter values using the same Go template syntax with `{{ .param_name }}`.

## Go Template Features

The MCP CLI Adapter uses Go's text/template package for parameter substitution, which supports a variety of powerful features:

### Basic Substitution

```console
{{ .param_name }}
```

### Conditional Logic

```console
{{ if .param_name }}value is present{{ else }}value is not present{{ end }}
```

### Nested Conditions

```console
{{ if .param1 }}
  {{ if .param2 }}both params exist{{ else }}only param1 exists{{ end }}
{{ else }}
  param1 doesn't exist
{{ end }}
```

### Comparison Operators

```console
{{ if eq .param "value" }}equal{{ end }}
{{ if ne .param "value" }}not equal{{ end }}
{{ if lt .param 5 }}less than{{ end }}
{{ if gt .param 5 }}greater than{{ end }}
```

### String Operations

```console
{{ .param | lower }}    <!-- lowercase -->
{{ .param | upper }}    <!-- uppercase -->
{{ .param | title }}    <!-- title case -->
```

For more advanced template features, refer to the [Go text/template documentation](https://pkg.go.dev/text/template).
