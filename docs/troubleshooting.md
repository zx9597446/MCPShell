# Troubleshooting

Some problems can arise when using this adapter:

## Model Compatibility

Not all LLM models can use tools. Model capabilities vary significantly:

- **Tool-capable models**: Claude 3 Opus/Sonnet/Haiku, GPT-4, GPT-3.5 Turbo,
  and some other recent LLMs can use tools.
- **Limited models**: Older models, smaller models, or those not trained with tool
  usage may ignore or fail to properly use MCP tools.
- **Different behaviors**: Even among tool-capable models, the frequency and
  effectiveness of tool usage varies.

If your LLM isn't using the tools you've configured:

- Confirm the model supports tool/function calling
- Try explicitly instructing the model to use specific tools
- Consider upgrading to a more capable model version

## General Issues

If you encounter other issues, try the following steps:

1. Make sure you're using the latest version of MCP CLI Adapter
2. Check the logs for any error messages
3. Verify your configuration files for syntax errors
4. Visit the [GitHub repository](https://github.com/inercia/mcp-cli-adapter) for
   known issues and solutions.

## Logging and Debugging

If you need to troubleshoot issues with the MCP CLI adapter:

1. **Enable detailed logging**: Start the adapter with the `--logfile` argument to
   capture detailed logs:

   ```console
   mcp-cli-adapter --logfile debug.log
   ```

2. **Inspect log output**: Review the generated log file for error messages,
   API responses, and adapter behavior:

   ```console
   tail -f debug.log
   ```

The log file will contain information about tool registrations, command executions, and potential error messages that can help identify the source of problems.

## Configuration

### Configuration Changes

When you make changes to your configuration files:

- **Restart the Cursor client**: After modifying your YAML config file
  or `.cursor/mcp.json`, you **must restart the Cursor client** for changes
  to take effect.
- **Check MCP Server**: Ensure the MCP server has restarted correctly after
  configuration changes.
- **Verify Tool Registration**: If tools aren't appearing, check that they're
  properly defined and that the server is running.

Common configuration issues:

- Path problems in the `.cursor/mcp.json` file
- Syntax errors in the YAML configuration
- Missing required fields in tool definitions

### Configuration Validation

When writing MCP tool configurations, syntax errors or constraint problems can be difficult to diagnose. The CLI adapter provides a validation command to verify your configurations:

#### Using the Validate Command

Use the `validate` command with the `--config` flag to check a single configuration file:

```console
mcp-cli-adapter validate --config examples/config.yaml
```

Successful validation will show each tool found and verified:

```console
[INFO] Validating MCP configuration
[INFO] Using configuration file: examples/config.yaml
[INFO] Found 7 tools in configuration
Validated tool: 'hello_world' (with 2 constraints)
Validated tool: 'weather' (with 3 constraints)
...
[INFO] Configuration validation successful
```

#### Interpreting Validation Errors

Validation errors include specific information about the issue:

```console
[ERROR] Failed to compile constraints for tool 'filesystem_usage': 
failed to compile constraint 'path.matches('^[a-zA-Z0-9/._\-]+$')': 
ERROR: Syntax error: token recognition error at: ''^[a-zA-Z0-9/._\-'
```

The error message points to the specific tool and constraint causing the problem, helping you quickly locate and fix the issue.

Properly validating your configurations before deployment can prevent runtime errors and ensure your MCP tools work correctly with LLMs.

#### Validating all the examples

For projects with multiple configuration files, use the `validate-examples` Makefile target:

```console
make validate-examples
```

This target validates all YAML files in the examples directory and stops on the first error.

#### Common Validation Issues

1. **Regex Pattern Escaping**: When using regular expressions in constraints, backslashes must be properly escaped:

   ```yaml
   # Incorrect:
   - "path.matches('^[a-zA-Z0-9/._\-]+$')"   
   
   # Correct:
   - "path.matches('^[a-zA-Z0-9/._\\\\-]+$')"
   ```

2. **Type Mismatches**: Ensure consistent numeric types in constraints:

   ```yaml
   # Error - mixing int and double:
   - "depth == 0 || (depth >= 1 && depth <= 3)"
   
   # Correct - consistent double values:
   - "depth == 0.0 || (depth >= 1.0 && depth <= 3.0)"
   ```

3. **Syntax Errors**: Check for missing quotes, incorrect indentation, or invalid YAML syntax.
