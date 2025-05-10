# Using the MCP-CLI adapter in Visual Studio Code

This guide explains how to set up and use the MCP-CLI adapter with Visual Studio Code.

## Prerequisites

- Visual Studio Code with GitHub Copilot
- MCP-CLI adapter installed (built from source or downloaded binary)

## Setup Instructions

To use MCP-CLI adapter with Visual Studio Code, follow these steps:

1. **Create your YAML configuration file** for the tools you want to expose (e.g., `mcp-cli.yaml`).

   ```yaml
   mcp:
     run:
       shell: bash
     tools:
       - name: "weather"
         description: "Get the weather for a location"
         params:
           location:
             type: string
             description: "The location to get weather for"
             required: true
         constraints:
           - "location.size() <= 50"  # Prevent overly long inputs
         run:
           command: "curl -s 'https://wttr.in/{{ .location }}?format=3'"
   ```

2. **Configure VS Code to use the MCP-CLI adapter** by creating a `.vscode/mcp.json` file in your workspace:

   ```json
   {
     "servers": {
       "mcp-cli-adapter": {
         "type": "stdio",
         "command": "/absolute/path/to/mcp-cli-adapter",
         "args": [
           "run", "--config", "/absolute/path/to/mcp-cli.yaml"
         ]
       }
     }
   }
   ```

   If you have Go installed, you can use it directly:

   ```json
   {
     "servers": {
       "mcp-cli-adapter": {
         "type": "stdio",
         "command": "go",
         "args": [
           "run", "github.com/inercia/mcp-cli-adapter",
           "run", "--config", "${workspaceFolder}/mcp-cli.yaml"
         ]
       }
     }
   }
   ```

   Note: You can use predefined VS Code variables like `${workspaceFolder}` in your configuration.

3. **Restart VS Code** or run the **MCP: List Servers** command from the Command Palette to start the server.

## Using Multiple MCP-CLI Adapter Instances

You can configure multiple instances of the MCP-CLI adapter,
each with different tool configurations:

```json
{
  "servers": {
    "mcp-cli-example": {
      "type": "stdio",
      "command": "/absolute/path/to/mcp-cli-adapter",
      "args": [
        "run",
        "--config", "${workspaceFolder}/examples/config.yaml",
        "--logfile", "${workspaceFolder}/debug.log"
      ]
    },
    "mcp-cli-kubernetes-ro": {
      "type": "stdio",
      "command": "/absolute/path/to/mcp-cli-adapter",
      "args": [
        "run",
        "--config", "${workspaceFolder}/examples/kubectl-ro.yaml",
        "--logfile", "${workspaceFolder}/debug.kubernetes-ro.log"
      ],
      "env": {
        "KUBECONFIG": "${workspaceFolder}/kubeconfig.yaml"
      }
    }
  }
}
```

## Setting up for Sensitive Information

If your tools require API keys or other sensitive information, you can use input variables:

```json
{
  "inputs": [
    {
      "type": "promptString",
      "id": "api-key",
      "description": "API Key",
      "password": true
    }
  ],
  "servers": {
    "mcp-cli-adapter": {
      "type": "stdio",
      "command": "/absolute/path/to/mcp-cli-adapter",
      "args": [
        "run", "--config", "${workspaceFolder}/mcp-cli.yaml"
      ],
      "env": {
        "API_KEY": "${input:api-key}"
      }
    }
  }
}
```

VS Code will prompt for these values when the server starts for the first time and securely store them for subsequent use.

## Using the Tools in Agent Mode

After configuring the MCP-CLI adapter:

1. Open the **Chat** view (⌃⌘I on macOS, Ctrl+Alt+I on Windows/Linux)
2. Select **Agent** mode from the dropdown
3. Click the **Tools** button to view and select available tools
4. Enter your query in the chat input box

When a tool is invoked, you'll need to confirm the action before it runs. You can choose to automatically confirm the specific tool for the current session, workspace, or all future invocations.

## Managing MCP Servers

To manage your MCP servers:

1. Run the **MCP: List Servers** command from the Command Palette
2. Select a server to start, stop, restart, view configuration, or view server logs

## Troubleshooting

If you're experiencing issues with the MCP-CLI adapter in VS Code:

1. **Check for error indicators** in the Chat view. Select the error notification and then **Show Output** to view server logs.
2. **Verify paths**: Ensure all file paths in your configuration are correct.
3. **Environment variables**: Make sure any required environment variables are properly set.
4. **Permissions**: Verify that the MCP-CLI adapter binary has execution permissions.
5. **Connection type**: Ensure the server connection type (`type: "stdio"`) is correctly specified.

## Security Considerations

When using MCP-CLI adapter with VS Code, be aware of the following security considerations:

- The tools you configure have the same system access permissions as VS Code.
- Be careful with tools that execute shell commands or access sensitive files.
- Use constraints to limit what your tools can do, especially when executing commands.
- Consider running VS Code with restricted permissions when using powerful tools. 