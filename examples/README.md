# MCP-CLI Adapter examples

This directpory contains some examples of configurations for running
different tools that can be used by your LLM.

## Creating your own scripts with Cursor

Most of the examples in this directory have been generated automatically
with Cursor. If you want to create your own toolkit, you can open the Cursor
chat and type something like.

```text
Please take a look at the examples found in
https://github.com/inercia/mcp-cli-adapter/tree/main/examples.
They are YAML files that define groups of tools that can be used by an LLM.
The configuration format is defined in
https://github.com/inercia/mcp-cli-adapter/blob/main/docs/configuration.md
Please create a new configuration file for running [YOUR COMMAND].
Add constraints in order to make the command execution safe,
checking paramters and so on.
Provide only read-only commands, do not allow the execution
of code with side effects.
```

Once that Cursor has generated a configuration file, run the
`mcp-cli-adapter validate` command in order to validate the file.
If it doesn't validate, pass the errors to Cursor (or allow
Cursor to run this command automatically). Cursor should be able
to fix these errors.

Please submit your toolkit to this repository if you consider
it useful for the community.
