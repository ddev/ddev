// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package mcp

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"iter"
	"maps"
	"net/url"
	"path/filepath"
	"reflect"
	"slices"
	"sync"
	"time"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/internal/jsonrpc2"
	"github.com/modelcontextprotocol/go-sdk/internal/util"
	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/yosida95/uritemplate/v3"
)

const DefaultPageSize = 1000

// A Server is an instance of an MCP server.
//
// Servers expose server-side MCP features, which can serve one or more MCP
// sessions by using [Server.Run].
type Server struct {
	// fixed at creation
	impl *Implementation
	opts ServerOptions

	mu                      sync.Mutex
	prompts                 *featureSet[*serverPrompt]
	tools                   *featureSet[*serverTool]
	resources               *featureSet[*serverResource]
	resourceTemplates       *featureSet[*serverResourceTemplate]
	sessions                []*ServerSession
	sendingMethodHandler_   MethodHandler
	receivingMethodHandler_ MethodHandler
	resourceSubscriptions   map[string]map[*ServerSession]bool // uri -> session -> bool
}

// ServerOptions is used to configure behavior of the server.
type ServerOptions struct {
	// Optional instructions for connected clients.
	Instructions string
	// If non-nil, called when "notifications/initialized" is received.
	InitializedHandler func(context.Context, *InitializedRequest)
	// PageSize is the maximum number of items to return in a single page for
	// list methods (e.g. ListTools).
	//
	// If zero, defaults to [DefaultPageSize].
	PageSize int
	// If non-nil, called when "notifications/roots/list_changed" is received.
	RootsListChangedHandler func(context.Context, *RootsListChangedRequest)
	// If non-nil, called when "notifications/progress" is received.
	ProgressNotificationHandler func(context.Context, *ProgressNotificationServerRequest)
	// If non-nil, called when "completion/complete" is received.
	CompletionHandler func(context.Context, *CompleteRequest) (*CompleteResult, error)
	// If non-zero, defines an interval for regular "ping" requests.
	// If the peer fails to respond to pings originating from the keepalive check,
	// the session is automatically closed.
	KeepAlive time.Duration
	// Function called when a client session subscribes to a resource.
	SubscribeHandler func(context.Context, *SubscribeRequest) error
	// Function called when a client session unsubscribes from a resource.
	UnsubscribeHandler func(context.Context, *UnsubscribeRequest) error
	// If true, advertises the prompts capability during initialization,
	// even if no prompts have been registered.
	HasPrompts bool
	// If true, advertises the resources capability during initialization,
	// even if no resources have been registered.
	HasResources bool
	// If true, advertises the tools capability during initialization,
	// even if no tools have been registered.
	HasTools bool
}

// NewServer creates a new MCP server. The resulting server has no features:
// add features using the various Server.AddXXX methods, and the [AddTool] function.
//
// The server can be connected to one or more MCP clients using [Server.Run].
//
// The first argument must not be nil.
//
// If non-nil, the provided options are used to configure the server.
func NewServer(impl *Implementation, options *ServerOptions) *Server {
	if impl == nil {
		panic("nil Implementation")
	}
	var opts ServerOptions
	if options != nil {
		opts = *options
	}
	options = nil // prevent reuse
	if opts.PageSize < 0 {
		panic(fmt.Errorf("invalid page size %d", opts.PageSize))
	}
	if opts.PageSize == 0 {
		opts.PageSize = DefaultPageSize
	}
	if opts.SubscribeHandler != nil && opts.UnsubscribeHandler == nil {
		panic("SubscribeHandler requires UnsubscribeHandler")
	}
	if opts.UnsubscribeHandler != nil && opts.SubscribeHandler == nil {
		panic("UnsubscribeHandler requires SubscribeHandler")
	}
	return &Server{
		impl:                    impl,
		opts:                    opts,
		prompts:                 newFeatureSet(func(p *serverPrompt) string { return p.prompt.Name }),
		tools:                   newFeatureSet(func(t *serverTool) string { return t.tool.Name }),
		resources:               newFeatureSet(func(r *serverResource) string { return r.resource.URI }),
		resourceTemplates:       newFeatureSet(func(t *serverResourceTemplate) string { return t.resourceTemplate.URITemplate }),
		sendingMethodHandler_:   defaultSendingMethodHandler[*ServerSession],
		receivingMethodHandler_: defaultReceivingMethodHandler[*ServerSession],
		resourceSubscriptions:   make(map[string]map[*ServerSession]bool),
	}
}

// AddPrompt adds a [Prompt] to the server, or replaces one with the same name.
func (s *Server) AddPrompt(p *Prompt, h PromptHandler) {
	// Assume there was a change, since add replaces existing items.
	// (It's possible an item was replaced with an identical one, but not worth checking.)
	s.changeAndNotify(
		notificationPromptListChanged,
		&PromptListChangedParams{},
		func() bool { s.prompts.add(&serverPrompt{p, h}); return true })
}

// RemovePrompts removes the prompts with the given names.
// It is not an error to remove a nonexistent prompt.
func (s *Server) RemovePrompts(names ...string) {
	s.changeAndNotify(notificationPromptListChanged, &PromptListChangedParams{},
		func() bool { return s.prompts.remove(names...) })
}

// AddTool adds a [Tool] to the server, or replaces one with the same name.
// The Tool argument must not be modified after this call.
//
// The tool's input schema must be non-nil and have the type "object". For a tool
// that takes no input, or one where any input is valid, set [Tool.InputSchema] to
// &jsonschema.Schema{Type: "object"}.
//
// If present, the output schema must also have type "object".
//
// When the handler is invoked as part of a CallTool request, req.Params.Arguments
// will be a json.RawMessage.
//
// Unmarshaling the arguments and validating them against the input schema are the
// caller's responsibility.
//
// Validating the result against the output schema, if any, is the caller's responsibility.
//
// Setting the result's Content, StructuredContent and IsError fields are the caller's
// responsibility.
//
// Most users should use the top-level function [AddTool], which handles all these
// responsibilities.
func (s *Server) AddTool(t *Tool, h ToolHandler) {
	if t.InputSchema == nil {
		// This prevents the tool author from forgetting to write a schema where
		// one should be provided. If we papered over this by supplying the empty
		// schema, then every input would be validated and the problem wouldn't be
		// discovered until runtime, when the LLM sent bad data.
		panic(fmt.Errorf("AddTool %q: missing input schema", t.Name))
	}
	if t.InputSchema.Type != "object" {
		panic(fmt.Errorf(`AddTool %q: input schema must have type "object"`, t.Name))
	}
	if t.OutputSchema != nil && t.OutputSchema.Type != "object" {
		panic(fmt.Errorf(`AddTool %q: output schema must have type "object"`, t.Name))
	}
	st := &serverTool{tool: t, handler: h}
	// Assume there was a change, since add replaces existing tools.
	// (It's possible a tool was replaced with an identical one, but not worth checking.)
	// TODO: Batch these changes by size and time? The typescript SDK doesn't.
	// TODO: Surface notify error here? best not, in case we need to batch.
	s.changeAndNotify(notificationToolListChanged, &ToolListChangedParams{},
		func() bool { s.tools.add(st); return true })
}

// TODO(v0.3.0): test
func toolForErr[In, Out any](t *Tool, h ToolHandlerFor[In, Out]) (*Tool, ToolHandler, error) {
	tt := *t

	// Special handling for an "any" input: treat as an empty object.
	if reflect.TypeFor[In]() == reflect.TypeFor[any]() && t.InputSchema == nil {
		tt.InputSchema = &jsonschema.Schema{Type: "object"}
	}

	var inputResolved *jsonschema.Resolved
	if _, err := setSchema[In](&tt.InputSchema, &inputResolved); err != nil {
		return nil, nil, fmt.Errorf("input schema: %w", err)
	}

	// Handling for zero values:
	//
	// If Out is a pointer type and we've derived the output schema from its
	// element type, use the zero value of its element type in place of a typed
	// nil.
	var (
		elemZero       any // only non-nil if Out is a pointer type
		outputResolved *jsonschema.Resolved
	)
	if t.OutputSchema != nil || reflect.TypeFor[Out]() != reflect.TypeFor[any]() {
		var err error
		elemZero, err = setSchema[Out](&tt.OutputSchema, &outputResolved)
		if err != nil {
			return nil, nil, fmt.Errorf("output schema: %v", err)
		}
	}

	th := func(ctx context.Context, req *CallToolRequest) (*CallToolResult, error) {
		// Unmarshal and validate args.
		var in In
		if req.Params.Arguments != nil {
			if err := unmarshalSchema(req.Params.Arguments, inputResolved, &in); err != nil {
				return nil, err
			}
		}

		// Call typed handler.
		res, out, err := h(ctx, req, in)
		// Handle server errors appropriately:
		// - If the handler returns a structured error (like jsonrpc2.WireError), return it directly
		// - If the handler returns a regular error, wrap it in a CallToolResult with IsError=true
		// - This allows tools to distinguish between protocol errors and tool execution errors
		if err != nil {
			// Check if this is already a structured JSON-RPC error
			if wireErr, ok := err.(*jsonrpc2.WireError); ok {
				return nil, wireErr
			}
			// For regular errors, embed them in the tool result as per MCP spec
			return &CallToolResult{
				Content: []Content{&TextContent{Text: err.Error()}},
				IsError: true,
			}, nil
		}

		// Validate output schema, if any.
		// Skip if out is nil: we've removed "null" from the output schema, so nil won't validate.
		if v := reflect.ValueOf(out); v.Kind() == reflect.Pointer && v.IsNil() {
		} else if err := validateSchema(outputResolved, &out); err != nil {
			return nil, fmt.Errorf("tool output: %w", err)
		}

		if res == nil {
			res = &CallToolResult{}
		}
		// Marshal the output and put the RawMessage in the StructuredContent field.
		var outval any = out
		if elemZero != nil {
			// Avoid typed nil, which will serialize as JSON null.
			// Instead, use the zero value of the unpointered type.
			var z Out
			if any(out) == any(z) { // zero is only non-nil if Out is a pointer type
				outval = elemZero
			}
		}
		if outval != nil {
			outbytes, err := json.Marshal(outval)
			if err != nil {
				return nil, fmt.Errorf("marshaling output: %w", err)
			}
			res.StructuredContent = json.RawMessage(outbytes) // avoid a second marshal over the wire

			// If the Content field isn't being used, return the serialized JSON in a
			// TextContent block, as the spec suggests:
			// https://modelcontextprotocol.io/specification/2025-06-18/server/tools#structured-content.
			if res.Content == nil {
				res.Content = []Content{&TextContent{
					Text: string(outbytes),
				}}
			}
		}
		return res, nil
	} // end of handler

	return &tt, th, nil
}

// setSchema sets the schema and resolved schema corresponding to the type T.
//
// If sfield is nil, the schema is derived from T.
//
// Pointers are treated equivalently to non-pointers when deriving the schema.
// If an indirection occurred to derive the schema, a non-nil zero value is
// returned to be used in place of the typed nil zero value.
//
// Note that if sfield already holds a schema, zero will be nil even if T is a
// pointer: if the user provided the schema, they may have intentionally
// derived it from the pointer type, and handling of zero values is up to them.
//
// TODO(rfindley): we really shouldn't ever return 'null' results. Maybe we
// should have a jsonschema.Zero(schema) helper?
func setSchema[T any](sfield **jsonschema.Schema, rfield **jsonschema.Resolved) (zero any, err error) {
	if *sfield == nil {
		rt := reflect.TypeFor[T]()
		if rt.Kind() == reflect.Pointer {
			rt = rt.Elem()
			zero = reflect.Zero(rt).Interface()
		}
		// TODO: we should be able to pass nil opts here.
		*sfield, err = jsonschema.ForType(rt, &jsonschema.ForOptions{})
	}
	if err != nil {
		return zero, err
	}
	*rfield, err = (*sfield).Resolve(&jsonschema.ResolveOptions{ValidateDefaults: true})
	return zero, err
}

// AddTool adds a tool and typed tool handler to the server.
//
// If the tool's input schema is nil, it is set to the schema inferred from the
// In type parameter, using [jsonschema.For]. The In type argument must be a
// map or a struct, so that its inferred JSON Schema has type "object".
//
// If the tool's output schema is nil, and the Out type is not 'any', the
// output schema is set to the schema inferred from the Out type argument,
// which also must be a map or struct.
//
// Unlike [Server.AddTool], AddTool does a lot automatically, and forces tools
// to conform to the MCP spec. See [ToolHandlerFor] for a detailed description
// of this automatic behavior.
func AddTool[In, Out any](s *Server, t *Tool, h ToolHandlerFor[In, Out]) {
	tt, hh, err := toolForErr(t, h)
	if err != nil {
		panic(fmt.Sprintf("AddTool: tool %q: %v", t.Name, err))
	}
	s.AddTool(tt, hh)
}

// RemoveTools removes the tools with the given names.
// It is not an error to remove a nonexistent tool.
func (s *Server) RemoveTools(names ...string) {
	s.changeAndNotify(notificationToolListChanged, &ToolListChangedParams{},
		func() bool { return s.tools.remove(names...) })
}

// AddResource adds a [Resource] to the server, or replaces one with the same URI.
// AddResource panics if the resource URI is invalid or not absolute (has an empty scheme).
func (s *Server) AddResource(r *Resource, h ResourceHandler) {
	s.changeAndNotify(notificationResourceListChanged, &ResourceListChangedParams{},
		func() bool {
			if _, err := url.Parse(r.URI); err != nil {
				panic(err) // url.Parse includes the URI in the error
			}
			s.resources.add(&serverResource{r, h})
			return true
		})
}

// RemoveResources removes the resources with the given URIs.
// It is not an error to remove a nonexistent resource.
func (s *Server) RemoveResources(uris ...string) {
	s.changeAndNotify(notificationResourceListChanged, &ResourceListChangedParams{},
		func() bool { return s.resources.remove(uris...) })
}

// AddResourceTemplate adds a [ResourceTemplate] to the server, or replaces one with the same URI.
// AddResourceTemplate panics if a URI template is invalid or not absolute (has an empty scheme).
func (s *Server) AddResourceTemplate(t *ResourceTemplate, h ResourceHandler) {
	s.changeAndNotify(notificationResourceListChanged, &ResourceListChangedParams{},
		func() bool {
			// Validate the URI template syntax
			_, err := uritemplate.New(t.URITemplate)
			if err != nil {
				panic(fmt.Errorf("URI template %q is invalid: %w", t.URITemplate, err))
			}
			s.resourceTemplates.add(&serverResourceTemplate{t, h})
			return true
		})
}

// RemoveResourceTemplates removes the resource templates with the given URI templates.
// It is not an error to remove a nonexistent resource.
func (s *Server) RemoveResourceTemplates(uriTemplates ...string) {
	s.changeAndNotify(notificationResourceListChanged, &ResourceListChangedParams{},
		func() bool { return s.resourceTemplates.remove(uriTemplates...) })
}

func (s *Server) capabilities() *ServerCapabilities {
	s.mu.Lock()
	defer s.mu.Unlock()

	caps := &ServerCapabilities{
		Logging: &LoggingCapabilities{},
	}
	if s.opts.HasTools || s.tools.len() > 0 {
		caps.Tools = &ToolCapabilities{ListChanged: true}
	}
	if s.opts.HasPrompts || s.prompts.len() > 0 {
		caps.Prompts = &PromptCapabilities{ListChanged: true}
	}
	if s.opts.HasResources || s.resources.len() > 0 || s.resourceTemplates.len() > 0 {
		caps.Resources = &ResourceCapabilities{ListChanged: true}
		if s.opts.SubscribeHandler != nil {
			caps.Resources.Subscribe = true
		}
	}
	if s.opts.CompletionHandler != nil {
		caps.Completions = &CompletionCapabilities{}
	}
	return caps
}

func (s *Server) complete(ctx context.Context, req *CompleteRequest) (*CompleteResult, error) {
	if s.opts.CompletionHandler == nil {
		return nil, jsonrpc2.ErrMethodNotFound
	}
	return s.opts.CompletionHandler(ctx, req)
}

// changeAndNotify is called when a feature is added or removed.
// It calls change, which should do the work and report whether a change actually occurred.
// If there was a change, it notifies a snapshot of the sessions.
func (s *Server) changeAndNotify(notification string, params Params, change func() bool) {
	var sessions []*ServerSession
	// Lock for the change, but not for the notification.
	s.mu.Lock()
	if change() {
		sessions = slices.Clone(s.sessions)
	}
	s.mu.Unlock()
	notifySessions(sessions, notification, params)
}

// Sessions returns an iterator that yields the current set of server sessions.
func (s *Server) Sessions() iter.Seq[*ServerSession] {
	s.mu.Lock()
	clients := slices.Clone(s.sessions)
	s.mu.Unlock()
	return slices.Values(clients)
}

func (s *Server) listPrompts(_ context.Context, req *ListPromptsRequest) (*ListPromptsResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if req.Params == nil {
		req.Params = &ListPromptsParams{}
	}
	return paginateList(s.prompts, s.opts.PageSize, req.Params, &ListPromptsResult{}, func(res *ListPromptsResult, prompts []*serverPrompt) {
		res.Prompts = []*Prompt{} // avoid JSON null
		for _, p := range prompts {
			res.Prompts = append(res.Prompts, p.prompt)
		}
	})
}

func (s *Server) getPrompt(ctx context.Context, req *GetPromptRequest) (*GetPromptResult, error) {
	s.mu.Lock()
	prompt, ok := s.prompts.get(req.Params.Name)
	s.mu.Unlock()
	if !ok {
		// Return a proper JSON-RPC error with the correct error code
		return nil, &jsonrpc2.WireError{
			Code:    CodeInvalidParams,
			Message: fmt.Sprintf("unknown prompt %q", req.Params.Name),
		}
	}
	return prompt.handler(ctx, req)
}

func (s *Server) listTools(_ context.Context, req *ListToolsRequest) (*ListToolsResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if req.Params == nil {
		req.Params = &ListToolsParams{}
	}
	return paginateList(s.tools, s.opts.PageSize, req.Params, &ListToolsResult{}, func(res *ListToolsResult, tools []*serverTool) {
		res.Tools = []*Tool{} // avoid JSON null
		for _, t := range tools {
			res.Tools = append(res.Tools, t.tool)
		}
	})
}

func (s *Server) callTool(ctx context.Context, req *CallToolRequest) (*CallToolResult, error) {
	s.mu.Lock()
	st, ok := s.tools.get(req.Params.Name)
	s.mu.Unlock()
	if !ok {
		return nil, &jsonrpc2.WireError{
			Code:    CodeInvalidParams,
			Message: fmt.Sprintf("unknown tool %q", req.Params.Name),
		}
	}
	res, err := st.handler(ctx, req)
	if err == nil && res != nil && res.Content == nil {
		res2 := *res
		res2.Content = []Content{} // avoid "null"
		res = &res2
	}
	return res, err
}

func (s *Server) listResources(_ context.Context, req *ListResourcesRequest) (*ListResourcesResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if req.Params == nil {
		req.Params = &ListResourcesParams{}
	}
	return paginateList(s.resources, s.opts.PageSize, req.Params, &ListResourcesResult{}, func(res *ListResourcesResult, resources []*serverResource) {
		res.Resources = []*Resource{} // avoid JSON null
		for _, r := range resources {
			res.Resources = append(res.Resources, r.resource)
		}
	})
}

func (s *Server) listResourceTemplates(_ context.Context, req *ListResourceTemplatesRequest) (*ListResourceTemplatesResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if req.Params == nil {
		req.Params = &ListResourceTemplatesParams{}
	}
	return paginateList(s.resourceTemplates, s.opts.PageSize, req.Params, &ListResourceTemplatesResult{},
		func(res *ListResourceTemplatesResult, rts []*serverResourceTemplate) {
			res.ResourceTemplates = []*ResourceTemplate{} // avoid JSON null
			for _, rt := range rts {
				res.ResourceTemplates = append(res.ResourceTemplates, rt.resourceTemplate)
			}
		})
}

func (s *Server) readResource(ctx context.Context, req *ReadResourceRequest) (*ReadResourceResult, error) {
	uri := req.Params.URI
	// Look up the resource URI in the lists of resources and resource templates.
	// This is a security check as well as an information lookup.
	handler, mimeType, ok := s.lookupResourceHandler(uri)
	if !ok {
		// Don't expose the server configuration to the client.
		// Treat an unregistered resource the same as a registered one that couldn't be found.
		return nil, ResourceNotFoundError(uri)
	}
	res, err := handler(ctx, req)
	if err != nil {
		return nil, err
	}
	if res == nil || res.Contents == nil {
		return nil, fmt.Errorf("reading resource %s: read handler returned nil information", uri)
	}
	// As a convenience, populate some fields.
	for _, c := range res.Contents {
		if c.URI == "" {
			c.URI = uri
		}
		if c.MIMEType == "" {
			c.MIMEType = mimeType
		}
	}
	return res, nil
}

// lookupResourceHandler returns the resource handler and MIME type for the resource or
// resource template matching uri. If none, the last return value is false.
func (s *Server) lookupResourceHandler(uri string) (ResourceHandler, string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Try resources first.
	if r, ok := s.resources.get(uri); ok {
		return r.handler, r.resource.MIMEType, true
	}
	// Look for matching template.
	for rt := range s.resourceTemplates.all() {
		if rt.Matches(uri) {
			return rt.handler, rt.resourceTemplate.MIMEType, true
		}
	}
	return nil, "", false
}

// fileResourceHandler returns a ReadResourceHandler that reads paths using dir as
// a base directory.
// It honors client roots and protects against path traversal attacks.
//
// The dir argument should be a filesystem path. It need not be absolute, but
// that is recommended to avoid a dependency on the current working directory (the
// check against client roots is done with an absolute path). If dir is not absolute
// and the current working directory is unavailable, fileResourceHandler panics.
//
// Lexical path traversal attacks, where the path has ".." elements that escape dir,
// are always caught. Go 1.24 and above also protects against symlink-based attacks,
// where symlinks under dir lead out of the tree.
func fileResourceHandler(dir string) ResourceHandler {
	// Convert dir to an absolute path.
	dirFilepath, err := filepath.Abs(dir)
	if err != nil {
		panic(err)
	}
	return func(ctx context.Context, req *ReadResourceRequest) (_ *ReadResourceResult, err error) {
		defer util.Wrapf(&err, "reading resource %s", req.Params.URI)

		// TODO(#25): use a memoizing API here.
		rootRes, err := req.Session.ListRoots(ctx, nil)
		if err != nil {
			return nil, fmt.Errorf("listing roots: %w", err)
		}
		roots, err := fileRoots(rootRes.Roots)
		if err != nil {
			return nil, err
		}
		data, err := readFileResource(req.Params.URI, dirFilepath, roots)
		if err != nil {
			return nil, err
		}
		// TODO(jba): figure out mime type. Omit for now: Server.readResource will fill it in.
		return &ReadResourceResult{Contents: []*ResourceContents{
			{URI: req.Params.URI, Blob: data},
		}}, nil
	}
}

// ResourceUpdated sends a notification to all clients that have subscribed to the
// resource specified in params. This method is the primary way for a
// server author to signal that a resource has changed.
func (s *Server) ResourceUpdated(ctx context.Context, params *ResourceUpdatedNotificationParams) error {
	s.mu.Lock()
	subscribedSessions := s.resourceSubscriptions[params.URI]
	sessions := slices.Collect(maps.Keys(subscribedSessions))
	s.mu.Unlock()
	notifySessions(sessions, notificationResourceUpdated, params)
	return nil
}

func (s *Server) subscribe(ctx context.Context, req *SubscribeRequest) (*emptyResult, error) {
	if s.opts.SubscribeHandler == nil {
		return nil, fmt.Errorf("%w: server does not support resource subscriptions", jsonrpc2.ErrMethodNotFound)
	}
	if err := s.opts.SubscribeHandler(ctx, req); err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.resourceSubscriptions[req.Params.URI] == nil {
		s.resourceSubscriptions[req.Params.URI] = make(map[*ServerSession]bool)
	}
	s.resourceSubscriptions[req.Params.URI][req.Session] = true

	return &emptyResult{}, nil
}

func (s *Server) unsubscribe(ctx context.Context, req *UnsubscribeRequest) (*emptyResult, error) {
	if s.opts.UnsubscribeHandler == nil {
		return nil, jsonrpc2.ErrMethodNotFound
	}

	if err := s.opts.UnsubscribeHandler(ctx, req); err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if subscribedSessions, ok := s.resourceSubscriptions[req.Params.URI]; ok {
		delete(subscribedSessions, req.Session)
		if len(subscribedSessions) == 0 {
			delete(s.resourceSubscriptions, req.Params.URI)
		}
	}

	return &emptyResult{}, nil
}

// Run runs the server over the given transport, which must be persistent.
//
// Run blocks until the client terminates the connection or the provided
// context is cancelled. If the context is cancelled, Run closes the connection.
//
// If tools have been added to the server before this call, then the server will
// advertise the capability for tools, including the ability to send list-changed notifications.
// If no tools have been added, the server will not have the tool capability.
// The same goes for other features like prompts and resources.
//
// Run is a convenience for servers that handle a single session (or one session at a time).
// It need not be called on servers that are used for multiple concurrent connections,
// as with [StreamableHTTPHandler].
func (s *Server) Run(ctx context.Context, t Transport) error {
	ss, err := s.Connect(ctx, t, nil)
	if err != nil {
		return err
	}

	ssClosed := make(chan error)
	go func() {
		ssClosed <- ss.Wait()
	}()

	select {
	case <-ctx.Done():
		ss.Close()
		return ctx.Err()
	case err := <-ssClosed:
		return err
	}
}

// bind implements the binder[*ServerSession] interface, so that Servers can
// be connected using [connect].
func (s *Server) bind(mcpConn Connection, conn *jsonrpc2.Connection, state *ServerSessionState, onClose func()) *ServerSession {
	assert(mcpConn != nil && conn != nil, "nil connection")
	ss := &ServerSession{conn: conn, mcpConn: mcpConn, server: s, onClose: onClose}
	if state != nil {
		ss.state = *state
	}
	s.mu.Lock()
	s.sessions = append(s.sessions, ss)
	s.mu.Unlock()
	return ss
}

// disconnect implements the binder[*ServerSession] interface, so that
// Servers can be connected using [connect].
func (s *Server) disconnect(cc *ServerSession) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions = slices.DeleteFunc(s.sessions, func(cc2 *ServerSession) bool {
		return cc2 == cc
	})

	for _, subscribedSessions := range s.resourceSubscriptions {
		delete(subscribedSessions, cc)
	}
}

// ServerSessionOptions configures the server session.
type ServerSessionOptions struct {
	State *ServerSessionState

	onClose func()
}

// Connect connects the MCP server over the given transport and starts handling
// messages.
//
// It returns a connection object that may be used to terminate the connection
// (with [Connection.Close]), or await client termination (with
// [Connection.Wait]).
//
// If opts.State is non-nil, it is the initial state for the server.
func (s *Server) Connect(ctx context.Context, t Transport, opts *ServerSessionOptions) (*ServerSession, error) {
	var state *ServerSessionState
	var onClose func()
	if opts != nil {
		state = opts.State
		onClose = opts.onClose
	}
	return connect(ctx, t, s, state, onClose)
}

// TODO: (nit) move all ServerSession methods below the ServerSession declaration.
func (ss *ServerSession) initialized(ctx context.Context, params *InitializedParams) (Result, error) {
	if params == nil {
		// Since we use nilness to signal 'initialized' state, we must ensure that
		// params are non-nil.
		params = new(InitializedParams)
	}
	var wasInit, wasInitd bool
	ss.updateState(func(state *ServerSessionState) {
		wasInit = state.InitializeParams != nil
		wasInitd = state.InitializedParams != nil
		if wasInit && !wasInitd {
			state.InitializedParams = params
		}
	})

	if !wasInit {
		return nil, fmt.Errorf("%q before %q", notificationInitialized, methodInitialize)
	}
	if wasInitd {
		return nil, fmt.Errorf("duplicate %q received", notificationInitialized)
	}
	if ss.server.opts.KeepAlive > 0 {
		ss.startKeepalive(ss.server.opts.KeepAlive)
	}
	if h := ss.server.opts.InitializedHandler; h != nil {
		h(ctx, serverRequestFor(ss, params))
	}
	return nil, nil
}

func (s *Server) callRootsListChangedHandler(ctx context.Context, req *RootsListChangedRequest) (Result, error) {
	if h := s.opts.RootsListChangedHandler; h != nil {
		h(ctx, req)
	}
	return nil, nil
}

func (ss *ServerSession) callProgressNotificationHandler(ctx context.Context, p *ProgressNotificationParams) (Result, error) {
	if h := ss.server.opts.ProgressNotificationHandler; h != nil {
		h(ctx, serverRequestFor(ss, p))
	}
	return nil, nil
}

// NotifyProgress sends a progress notification from the server to the client
// associated with this session.
// This is typically used to report on the status of a long-running request
// that was initiated by the client.
func (ss *ServerSession) NotifyProgress(ctx context.Context, params *ProgressNotificationParams) error {
	return handleNotify(ctx, notificationProgress, newServerRequest(ss, orZero[Params](params)))
}

func newServerRequest[P Params](ss *ServerSession, params P) *ServerRequest[P] {
	return &ServerRequest[P]{Session: ss, Params: params}
}

// A ServerSession is a logical connection from a single MCP client. Its
// methods can be used to send requests or notifications to the client. Create
// a session by calling [Server.Connect].
//
// Call [ServerSession.Close] to close the connection, or await client
// termination with [ServerSession.Wait].
type ServerSession struct {
	onClose func()

	server          *Server
	conn            *jsonrpc2.Connection
	mcpConn         Connection
	keepaliveCancel context.CancelFunc // TODO: theory around why keepaliveCancel need not be guarded

	mu    sync.Mutex
	state ServerSessionState
}

func (ss *ServerSession) updateState(mut func(*ServerSessionState)) {
	ss.mu.Lock()
	mut(&ss.state)
	copy := ss.state
	ss.mu.Unlock()
	if c, ok := ss.mcpConn.(serverConnection); ok {
		c.sessionUpdated(copy)
	}
}

// hasInitialized reports whether the server has received the initialized
// notification.
//
// TODO(findleyr): use this to prevent change notifications.
func (ss *ServerSession) hasInitialized() bool {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	return ss.state.InitializedParams != nil
}

// checkInitialized returns a formatted error if the server has not yet
// received the initialized notification.
func (ss *ServerSession) checkInitialized(method string) error {
	if !ss.hasInitialized() {
		// TODO(rfindley): enable this check.
		// Right now is is flaky, because server tests don't await the initialized notification.
		// Perhaps requests should simply block until they have received the initialized notification

		// if strings.HasPrefix(method, "notifications/") {
		// 	return fmt.Errorf("must not send %q before %q is received", method, notificationInitialized)
		// } else {
		// 	return fmt.Errorf("cannot call %q before %q is received", method, notificationInitialized)
		// }
	}
	return nil
}

func (ss *ServerSession) ID() string {
	if c, ok := ss.mcpConn.(hasSessionID); ok {
		return c.SessionID()
	}
	return ""
}

// Ping pings the client.
func (ss *ServerSession) Ping(ctx context.Context, params *PingParams) error {
	_, err := handleSend[*emptyResult](ctx, methodPing, newServerRequest(ss, orZero[Params](params)))
	return err
}

// ListRoots lists the client roots.
func (ss *ServerSession) ListRoots(ctx context.Context, params *ListRootsParams) (*ListRootsResult, error) {
	if err := ss.checkInitialized(methodListRoots); err != nil {
		return nil, err
	}
	return handleSend[*ListRootsResult](ctx, methodListRoots, newServerRequest(ss, orZero[Params](params)))
}

// CreateMessage sends a sampling request to the client.
func (ss *ServerSession) CreateMessage(ctx context.Context, params *CreateMessageParams) (*CreateMessageResult, error) {
	if err := ss.checkInitialized(methodCreateMessage); err != nil {
		return nil, err
	}
	if params == nil {
		params = &CreateMessageParams{Messages: []*SamplingMessage{}}
	}
	if params.Messages == nil {
		p2 := *params
		p2.Messages = []*SamplingMessage{} // avoid JSON "null"
		params = &p2
	}
	return handleSend[*CreateMessageResult](ctx, methodCreateMessage, newServerRequest(ss, orZero[Params](params)))
}

// Elicit sends an elicitation request to the client asking for user input.
func (ss *ServerSession) Elicit(ctx context.Context, params *ElicitParams) (*ElicitResult, error) {
	if err := ss.checkInitialized(methodElicit); err != nil {
		return nil, err
	}
	return handleSend[*ElicitResult](ctx, methodElicit, newServerRequest(ss, orZero[Params](params)))
}

// Log sends a log message to the client.
// The message is not sent if the client has not called SetLevel, or if its level
// is below that of the last SetLevel.
func (ss *ServerSession) Log(ctx context.Context, params *LoggingMessageParams) error {
	ss.mu.Lock()
	logLevel := ss.state.LogLevel
	ss.mu.Unlock()
	if logLevel == "" {
		// The spec is unclear, but seems to imply that no log messages are sent until the client
		// sets the level.
		// TODO(jba): read other SDKs, possibly file an issue.
		return nil
	}
	if compareLevels(params.Level, logLevel) < 0 {
		return nil
	}
	return handleNotify(ctx, notificationLoggingMessage, newServerRequest(ss, orZero[Params](params)))
}

// AddSendingMiddleware wraps the current sending method handler using the provided
// middleware. Middleware is applied from right to left, so that the first one is
// executed first.
//
// For example, AddSendingMiddleware(m1, m2, m3) augments the method handler as
// m1(m2(m3(handler))).
//
// Sending middleware is called when a request is sent. It is useful for tasks
// such as tracing, metrics, and adding progress tokens.
func (s *Server) AddSendingMiddleware(middleware ...Middleware) {
	s.mu.Lock()
	defer s.mu.Unlock()
	addMiddleware(&s.sendingMethodHandler_, middleware)
}

// AddReceivingMiddleware wraps the current receiving method handler using
// the provided middleware. Middleware is applied from right to left, so that the
// first one is executed first.
//
// For example, AddReceivingMiddleware(m1, m2, m3) augments the method handler as
// m1(m2(m3(handler))).
//
// Receiving middleware is called when a request is received. It is useful for tasks
// such as authentication, request logging and metrics.
func (s *Server) AddReceivingMiddleware(middleware ...Middleware) {
	s.mu.Lock()
	defer s.mu.Unlock()
	addMiddleware(&s.receivingMethodHandler_, middleware)
}

// serverMethodInfos maps from the RPC method name to serverMethodInfos.
//
// The 'allowMissingParams' values are extracted from the protocol schema.
// TODO(rfindley): actually load and validate the protocol schema, rather than
// curating these method flags.
var serverMethodInfos = map[string]methodInfo{
	methodComplete:               newServerMethodInfo(serverMethod((*Server).complete), 0),
	methodInitialize:             newServerMethodInfo(serverSessionMethod((*ServerSession).initialize), 0),
	methodPing:                   newServerMethodInfo(serverSessionMethod((*ServerSession).ping), missingParamsOK),
	methodListPrompts:            newServerMethodInfo(serverMethod((*Server).listPrompts), missingParamsOK),
	methodGetPrompt:              newServerMethodInfo(serverMethod((*Server).getPrompt), 0),
	methodListTools:              newServerMethodInfo(serverMethod((*Server).listTools), missingParamsOK),
	methodCallTool:               newServerMethodInfo(serverMethod((*Server).callTool), 0),
	methodListResources:          newServerMethodInfo(serverMethod((*Server).listResources), missingParamsOK),
	methodListResourceTemplates:  newServerMethodInfo(serverMethod((*Server).listResourceTemplates), missingParamsOK),
	methodReadResource:           newServerMethodInfo(serverMethod((*Server).readResource), 0),
	methodSetLevel:               newServerMethodInfo(serverSessionMethod((*ServerSession).setLevel), 0),
	methodSubscribe:              newServerMethodInfo(serverMethod((*Server).subscribe), 0),
	methodUnsubscribe:            newServerMethodInfo(serverMethod((*Server).unsubscribe), 0),
	notificationCancelled:        newServerMethodInfo(serverSessionMethod((*ServerSession).cancel), notification|missingParamsOK),
	notificationInitialized:      newServerMethodInfo(serverSessionMethod((*ServerSession).initialized), notification|missingParamsOK),
	notificationRootsListChanged: newServerMethodInfo(serverMethod((*Server).callRootsListChangedHandler), notification|missingParamsOK),
	notificationProgress:         newServerMethodInfo(serverSessionMethod((*ServerSession).callProgressNotificationHandler), notification),
}

func (ss *ServerSession) sendingMethodInfos() map[string]methodInfo { return clientMethodInfos }

func (ss *ServerSession) receivingMethodInfos() map[string]methodInfo { return serverMethodInfos }

func (ss *ServerSession) sendingMethodHandler() MethodHandler {
	s := ss.server
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sendingMethodHandler_
}

func (ss *ServerSession) receivingMethodHandler() MethodHandler {
	s := ss.server
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.receivingMethodHandler_
}

// getConn implements [session.getConn].
func (ss *ServerSession) getConn() *jsonrpc2.Connection { return ss.conn }

// handle invokes the method described by the given JSON RPC request.
func (ss *ServerSession) handle(ctx context.Context, req *jsonrpc.Request) (any, error) {
	ss.mu.Lock()
	initialized := ss.state.InitializeParams != nil
	ss.mu.Unlock()

	// From the spec:
	// "The client SHOULD NOT send requests other than pings before the server
	// has responded to the initialize request."
	switch req.Method {
	case methodInitialize, methodPing, notificationInitialized:
	default:
		if !initialized {
			return nil, fmt.Errorf("method %q is invalid during session initialization", req.Method)
		}
	}

	// modelcontextprotocol/go-sdk#26: handle calls asynchronously, and
	// notifications synchronously, except for 'initialize' which shouldn't be
	// asynchronous to other
	if req.IsCall() && req.Method != methodInitialize {
		jsonrpc2.Async(ctx)
	}

	// For the streamable transport, we need the request ID to correlate
	// server->client calls and notifications to the incoming request from which
	// they originated. See [idContextKey] for details.
	ctx = context.WithValue(ctx, idContextKey{}, req.ID)
	return handleReceive(ctx, ss, req)
}

func (ss *ServerSession) InitializeParams() *InitializeParams { return ss.state.InitializeParams }

func (ss *ServerSession) initialize(ctx context.Context, params *InitializeParams) (*InitializeResult, error) {
	if params == nil {
		return nil, fmt.Errorf("%w: \"params\" must be be provided", jsonrpc2.ErrInvalidParams)
	}
	ss.updateState(func(state *ServerSessionState) {
		state.InitializeParams = params
	})

	s := ss.server
	return &InitializeResult{
		// TODO(rfindley): alter behavior when falling back to an older version:
		// reject unsupported features.
		ProtocolVersion: negotiatedVersion(params.ProtocolVersion),
		Capabilities:    s.capabilities(),
		Instructions:    s.opts.Instructions,
		ServerInfo:      s.impl,
	}, nil
}

func (ss *ServerSession) ping(context.Context, *PingParams) (*emptyResult, error) {
	return &emptyResult{}, nil
}

// cancel is a placeholder: cancellation is handled the jsonrpc2 package.
//
// It should never be invoked in practice because cancellation is preempted,
// but having its signature here facilitates the construction of methodInfo
// that can be used to validate incoming cancellation notifications.
func (ss *ServerSession) cancel(context.Context, *CancelledParams) (Result, error) {
	return nil, nil
}

func (ss *ServerSession) setLevel(_ context.Context, params *SetLoggingLevelParams) (*emptyResult, error) {
	ss.updateState(func(state *ServerSessionState) {
		state.LogLevel = params.Level
	})
	return &emptyResult{}, nil
}

// Close performs a graceful shutdown of the connection, preventing new
// requests from being handled, and waiting for ongoing requests to return.
// Close then terminates the connection.
func (ss *ServerSession) Close() error {
	if ss.keepaliveCancel != nil {
		// Note: keepaliveCancel access is safe without a mutex because:
		// 1. keepaliveCancel is only written once during startKeepalive (happens-before all Close calls)
		// 2. context.CancelFunc is safe to call multiple times and from multiple goroutines
		// 3. The keepalive goroutine calls Close on ping failure, but this is safe since
		//    Close is idempotent and conn.Close() handles concurrent calls correctly
		ss.keepaliveCancel()
	}
	err := ss.conn.Close()

	if ss.onClose != nil {
		ss.onClose()
	}

	return err
}

// Wait waits for the connection to be closed by the client.
func (ss *ServerSession) Wait() error {
	return ss.conn.Wait()
}

// startKeepalive starts the keepalive mechanism for this server session.
func (ss *ServerSession) startKeepalive(interval time.Duration) {
	startKeepalive(ss, interval, &ss.keepaliveCancel)
}

// pageToken is the internal structure for the opaque pagination cursor.
// It will be Gob-encoded and then Base64-encoded for use as a string token.
type pageToken struct {
	LastUID string // The unique ID of the last resource seen.
}

// encodeCursor encodes a unique identifier (UID) into a opaque pagination cursor
// by serializing a pageToken struct.
func encodeCursor(uid string) (string, error) {
	var buf bytes.Buffer
	token := pageToken{LastUID: uid}
	encoder := gob.NewEncoder(&buf)
	if err := encoder.Encode(token); err != nil {
		return "", fmt.Errorf("failed to encode page token: %w", err)
	}
	return base64.URLEncoding.EncodeToString(buf.Bytes()), nil
}

// decodeCursor decodes an opaque pagination cursor into the original pageToken struct.
func decodeCursor(cursor string) (*pageToken, error) {
	decodedBytes, err := base64.URLEncoding.DecodeString(cursor)
	if err != nil {
		return nil, fmt.Errorf("failed to decode cursor: %w", err)
	}

	var token pageToken
	buf := bytes.NewBuffer(decodedBytes)
	decoder := gob.NewDecoder(buf)
	if err := decoder.Decode(&token); err != nil {
		return nil, fmt.Errorf("failed to decode page token: %w, cursor: %v", err, cursor)
	}
	return &token, nil
}

// paginateList is a generic helper that returns a paginated slice of items
// from a featureSet. It populates the provided result res with the items
// and sets its next cursor for subsequent pages.
// If there are no more pages, the next cursor within the result will be an empty string.
func paginateList[P listParams, R listResult[T], T any](fs *featureSet[T], pageSize int, params P, res R, setFunc func(R, []T)) (R, error) {
	var seq iter.Seq[T]
	if params.cursorPtr() == nil || *params.cursorPtr() == "" {
		seq = fs.all()
	} else {
		pageToken, err := decodeCursor(*params.cursorPtr())
		// According to the spec, invalid cursors should return Invalid params.
		if err != nil {
			var zero R
			return zero, jsonrpc2.ErrInvalidParams
		}
		seq = fs.above(pageToken.LastUID)
	}
	var count int
	var features []T
	for f := range seq {
		count++
		// If we've seen pageSize + 1 elements, we've gathered enough info to determine
		// if there's a next page. Stop processing the sequence.
		if count == pageSize+1 {
			break
		}
		features = append(features, f)
	}
	setFunc(res, features)
	// No remaining pages.
	if count < pageSize+1 {
		return res, nil
	}
	nextCursor, err := encodeCursor(fs.uniqueID(features[len(features)-1]))
	if err != nil {
		var zero R
		return zero, err
	}
	*res.nextCursorPtr() = nextCursor
	return res, nil
}
