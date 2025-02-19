package main

import (
	"context"
	"fmt"
	"log"
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
	}

	sessionTool := mcp.NewTool("new_session",
		mcp.WithDescription("Create a new session, any generated files are stored in the session"),
	)
	s.mcpServer.AddTool(sessionTool, s.newSessionHandler)

	execTool := mcp.NewTool("exec",
		mcp.WithDescription("Execute Python code in a session"),
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
			mcp.Description("Optional: Session ID to execute code in"),
		),
	)
	s.mcpServer.AddTool(listFilesTool, s.listFilesHandler)

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

// Start starts the server.
func (s *Server) Start() error {
	s.sessionID = xid.New().String()
	return server.ServeStdio(s.mcpServer)
}
