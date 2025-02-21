# Azure Code Interpreter MCP Server

This [MCP](https://modelcontextprotocol.io/introduction) server gives LLMs the ability to run Python code in a secure and sandboxed environment. The server is built on top of the [Azure Code Interpreter](https://learn.microsoft.com/en-us/azure/container-apps/sessions-code-interpreter) and gives the LLM the ability to execute code and download generated files in sessions that are ephemeral and isolated from the rest of the system.

## Setup

The server uses the [az]() Azure CLI client for authentication, so make sure that is installed and authenticated. You'll need to setup a resource group and session pool according to [these instructions](https://learn.microsoft.com/en-us/azure/container-apps/sessions-code-interpreter#code-interpreter-session-pool). Be sure to set the `--network-status EgressEnabled` flag when creating the session pool if you want generated code to access the internet.

You need to build the `azure-code-interpreter-mcp` binary with `go` and install it in your `$PATH` or reference it directly in your LLM tool.

## Usage

Once you have created the requred Azure resources and authenticated with `az`, you need to set the following environment variables in you LLM tool:

- `AZURE_SUBSCRIPTION_ID`: The ID of the Azure subscription that contains the resource group.
- `AZURE_RESOURCE_GROUP`: The name of the resource group that contains the session pool.
- `AZURE_SESSION_POOL`: The name of the session pool that the server should use.
- `AZURE_REGION`: The region that the session pool is in.
- `AZURE_DOWNLOAD_DIRECTORY`: The directory where the server should save downloaded files. This directory must be writable by the server process.

## Supported Server Functions

The LLM has access to the following functionality.

### New Session

The LLM can create a new session as it sees fit for deliniating tasks.

### Execute

Execute Python code in the context of the session. The LLM is prompted to save any created files to the `/mnt/data` directory, which Azure has setup to allow files to be downloaded.

### List Files

This will list the files that have been stored in the `/mnt/data` directory by the executed code.

### Download Files

Download the files from `/mnt/data` to the directory specified by `AZURE_DOWNLOAD_DIRECTORY` on the server.
