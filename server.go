package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rs/xid"
)

// Server contains the MCP Server implementation and session state.
type Server struct {
	mcpServer *server.MCPServer
	sessionID string
	azClient  *azureClient
	config    config
}

// NewServer creates a new Server.
func NewServer(cfg config) *Server {
	ms := server.NewMCPServer(
		"Azure Code Interpreter MCP Server",
		"0.0.1",
		server.WithLogging(),
	)
	ac, err := newAzureClient(cfg)
	if err != nil {
		log.Fatalf("failed to create Azure client: %v", err)
	}
	s := &Server{
		mcpServer: ms,
		azClient:  ac,
		config:    cfg,
	}

	sessionTool := mcp.NewTool("new_session",
		mcp.WithDescription("Create a new session, any generated files are stored in the session"),
		mcp.WithString("reason",
			mcp.Description("Optional: reason the session was created"),
		),
	)
	s.mcpServer.AddTool(sessionTool, s.newSessionHandler)

	execTool := mcp.NewTool("exec",
		mcp.WithDescription("Execute Python code in a session. Store all files generated in the `/mnt/data/` directory"),
		mcp.WithString("code",
			mcp.Required(),
			mcp.Description("Python code to execute in the session"),
		),
		mcp.WithString("session_id",
			mcp.Description("Optional: Session ID to execute code in"),
		),
	)
	s.mcpServer.AddTool(execTool, s.execHandler)

	listFilesTool := mcp.NewTool("list_files",
		mcp.WithDescription("List files in the session"),
		mcp.WithString("session_id",
			mcp.Description("Optional: Session ID to list files in"),
		),
	)
	s.mcpServer.AddTool(listFilesTool, s.listFilesHandler)

	downloadFileTool := mcp.NewTool("download_file",
		mcp.WithDescription("Download a file from the session to the local computer"),
		mcp.WithString("session_id",
			mcp.Description("Optional: Session ID to download file from"),
		),
		mcp.WithString("file_name",
			mcp.Required(),
			mcp.Description("Path of file to download from the session"),
		),
	)
	s.mcpServer.AddTool(downloadFileTool, s.downloadFileHandler)

	return s
}

func (s *Server) newSessionHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	s.sessionID = xid.New().String()
	return mcp.NewToolResultText(s.sessionID), nil
}

func (s *Server) execHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionID, ok := request.Params.Arguments["session_id"].(string)
	if !ok {
		sessionID = s.sessionID
	}
	code := request.Params.Arguments["code"].(string)
	code, _ = strings.CutPrefix(code, "```python")
	code, _ = strings.CutPrefix(code, "```")
	code, _ = strings.CutSuffix(code, "```\r\n")
	code, _ = strings.CutSuffix(code, "```\n")
	code, _ = strings.CutSuffix(code, "```")
	resp, err := s.azClient.execute(sessionID, code)
	if err != nil {
		return nil, fmt.Errorf("failed to execute code: %w", err)
	}
	if resp == "" {
		return mcp.NewToolResultText("No output"), nil
	}
	return mcp.NewToolResultText(resp), nil
}

func (s *Server) listFilesHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionID, ok := request.Params.Arguments["session_id"].(string)
	if !ok {
		sessionID = s.sessionID
	}
	resp, err := s.azClient.listFiles(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}
	return mcp.NewToolResultText(resp), nil
}

func (s *Server) downloadFileHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionID, ok := request.Params.Arguments["session_id"].(string)
	if !ok {
		sessionID = s.sessionID
	}
	fileName, _ := request.Params.Arguments["file_name"].(string)
	b, err := s.azClient.getFile(sessionID, fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to get file %s: %w", fileName, err)
	}
	dir := s.config.DownloadDirectory
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			return nil, fmt.Errorf("failed to create directory: %w", err)
		}
	}
	fp := filepath.Join(dir, fileName)
	err = os.WriteFile(fp, b, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}
	rc := mcp.BlobResourceContents{
		URI:  "file://" + fp,
		Blob: base64.StdEncoding.EncodeToString(b),
	}
	return mcp.NewToolResultResource(fileName, rc), nil
}

// Start starts the server.
func (s *Server) Start() error {
	s.sessionID = xid.New().String()
	return server.ServeStdio(s.mcpServer)
}
