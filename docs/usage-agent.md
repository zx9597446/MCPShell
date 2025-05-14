# MCPShell Agent Mode

The MCPShell can be run in "agent mode" to establish a direct connection between Large Language Models (LLMs) and the tools you define in your configuration. This allows for autonomous execution of tasks without requiring a separate MCP client like Cursor or Visual Studio Code.

## Overview

In agent mode, MCPShell:

1. Connects directly to an LLM API (currently OpenAI-compatible APIs)
2. Makes your tools available to the LLM
3. Manages the conversation flow 
4. Handles tool execution requests
5. Provides the tool results back to the LLM

This creates a complete AI agent that can perform tasks on your system using your defined tools.

## Command-Line Options

Run the agent with:

```bash
mcpshell agent [flags]
```

### Required Flags

- `--config`, `-c`: Path to the YAML configuration file (required)
- `--model`, `-m`: LLM model to use (e.g., "gpt-4o", "llama3", etc.)

### Optional Flags

- `--logfile`, `-l`: Path to the log file
- `--log-level`: Logging level (none, error, info, debug)
- `--system-prompt`, `-s`: System prompt for the LLM
- `--user-prompt`, `-u`: Initial user prompt for the LLM
- `--openai-api-key`, `-k`: OpenAI API key (or set OPENAI_API_KEY environment variable)
- `--openai-api-url`, `-b`: Base URL for the OpenAI API (for non-OpenAI services)
- `--once`, `-o`: Exit after receiving a final response (one-shot mode)

## Configuration File Extensions for Agent Mode

The standard MCPShell configuration file format has been extended to support agent-specific features, particularly pre-defined prompts:

```yaml
# New "prompts" section for agent mode
prompts:
  - system:
    - "You are a system administrator assistant."
    - "Use the available tools to help diagnose and solve problems."
    user:
    - "Help me analyze disk usage on my system."

# Standard MCPShell configuration
mcp:
  description: "Tools for system administration tasks"
  run:
    shell: "bash"
  tools:
    # Tool definitions...
```

### Prompts Section

The `prompts` section lets you define system and user prompts directly in the configuration file:

- **System Prompts**: Define the role and capabilities of the assistant
- **User Prompts**: Initial questions or instructions for the assistant

When you run without the `--system-prompt` or `--user-prompt` flags, MCPShell will use these prompts from the configuration file. Multiple prompts in each category will be joined with newlines.

## Creating an Agent Script

You can create a shell script to run MCPShell in agent mode with a specific configuration. This is useful for creating specialized agents for different tasks.

Here's a practical example of a configuration file for a disk space analyzer agent:

```yaml
prompts:
  - system:
    - "You are a disk space analyzer assistant that helps users identify what's consuming disk space."
    - "Always provide clear, step-by-step analysis of disk usage patterns."
    - "Suggest practical ways to free up space when appropriate."

mcp:
  description: "Tools for analyzing disk space usage and system performance"
  run:
    shell: "bash"
  tools:
    - name: "disk_usage"
      description: "Check disk usage for a directory"
      params:
        directory:
          type: string
          description: "Directory to analyze"
          required: true
        max_depth:
          type: number
          description: "Maximum depth to analyze (1-3)"
          default: 2
      constraints:
        - "directory.startsWith('/')"  # Must be absolute path
        - "!directory.contains('..')"  # Prevent directory traversal
        - "max_depth >= 1 && max_depth <= 3"  # Limit recursion depth
      run:
        command: |
          du -h --max-depth={{ .max_depth }} {{ .directory }} | sort -hr | head -20
      output:
        format: "text"
        prefix: "Disk Usage Analysis (Top 20 largest directories):"

    - name: "filesystem_info"
      description: "Show filesystem usage information"
      run:
        command: |
          df -h
      output:
        format: "text"
        prefix: "Filesystem Usage Information:"

    - name: "large_files"
      description: "Find large files in a directory"
      params:
        directory:
          type: string
          description: "Directory to search in"
          required: true
        min_size:
          type: string
          description: "Minimum file size (e.g., '100M', '1G')"
          default: "100M"
        file_type:
          type: string
          description: "Filter by file extension (e.g., 'log', 'zip')"
          required: false
      constraints:
        - "directory.startsWith('/')"  # Must be absolute path
        - "!directory.contains('..')"  # Prevent directory traversal
      run:
        command: |
          file_type="{{ .file_type }}"
          if [ -n "$file_type" ]; then
            find {{ .directory }} -type f -name "*.${file_type}" -size +{{ .min_size }} -exec ls -lh {} \; | sort -k5hr | head -20
          else
            find {{ .directory }} -type f -size +{{ .min_size }} -exec ls -lh {} \; | sort -k5hr | head -20
          fi
      output:
        format: "text"
        prefix: "Large Files (minimum size {{ .min_size }}):"
```

## Running the Agent

With the configuration above saved as `disk-analyzer.yaml`, you can run:

```bash
mcpshell agent \
  --config disk-analyzer.yaml \
  --model "gpt-4o" \
  --openai-api-key "your-api-key"
  --user-prompt "My root partition is running low on space. Can you help me find what's taking up space and how I might free some up?"
```

The agent will:

1. Connect to OpenAI's API
2. Send the system and user prompts from the configuration
3. Make the `disk_usage`, `filesystem_info`, and `large_files` tools available
4. Process the LLM's responses and execute tool calls as requested

## Interacting with the Agent

In interactive mode (without the `--once` flag), the agent will:

- Display the LLM's responses
- Execute tool calls as requested by the LLM
- Prompt you for additional input
- Continue the conversation until you exit (Ctrl+C) or the LLM responds with "TERMINATE"

In one-shot mode (with the `--once` flag), the agent will:

- Process the initial prompt
- Execute any requested tools
- Display the final response
- Exit automatically

## Using with Local LLMs

MCPShell agent mode works well with local LLMs through services like Ollama:

```bash
mcpshell agent \
  --config disk-analyzer.yaml \
  --model "llama3" \
  --openai-api-key "ollama" \
  --openai-api-url "http://localhost:11434/v1"
```

NOTE: make sure you choose a model that accepts "tools".

## Testing and Debugging

When developing agents, you can:

1. Enable debug logging with `--log-level debug`
2. Examine the log file for detailed information
3. Test with the `exe` command to verify individual tools:
   ```bash
   mcpshell exe --config disk-analyzer.yaml disk_usage directory="/" max_depth=2
   ```

## Conclusion

The agent mode provides a powerful way to create specialized AI assistants that can perform specific tasks on your system using the tools you define. By combining well-defined tools with appropriate system and user prompts, you can create agents that solve real-world problems in a secure and controlled manner. 