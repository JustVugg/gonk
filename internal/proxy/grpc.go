// grpc.go - Same as original with corrected imports
package proxy

import (
    "context"
    "encoding/binary"
    "fmt"
    "io"
    "net/http"
    "strings"
    "sync"
    "time"
    
    "google.golang.org/grpc"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/credentials/insecure"
    "google.golang.org/grpc/keepalive"
    "google.golang.org/grpc/metadata"
    "google.golang.org/grpc/status"
)

type gRPCProxy struct {
    target   string
    conn     *grpc.ClientConn
    connMu   sync.RWMutex
    director func(*http.Request)
}

func newGRPCProxy(target string, director func(*http.Request)) (*gRPCProxy, error) {
    p := &gRPCProxy{
        target:   target,
        director: director,
    }
    
    if err := p.ensureConnection(); err != nil {
        return nil, err
    }
    
    return p, nil
}

func (p *gRPCProxy) ensureConnection() error {
    p.connMu.Lock()
    defer p.connMu.Unlock()
    
    if p.conn != nil {
        return nil
    }
    
    opts := []grpc.DialOption{
        grpc.WithTransportCredentials(insecure.NewCredentials()),
        grpc.WithDefaultCallOptions(
            grpc.MaxCallRecvMsgSize(16 * 1024 * 1024),
            grpc.MaxCallSendMsgSize(16 * 1024 * 1024),
        ),
        grpc.WithKeepaliveParams(keepalive.ClientParameters{
            Time:                10 * time.Second,
            Timeout:             time.Second,
            PermitWithoutStream: true,
        }),
    }
    
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    
    conn, err := grpc.DialContext(ctx, p.target, opts...)
    if err != nil {
        return fmt.Errorf("failed to dial gRPC: %w", err)
    }
    
    p.conn = conn
    return nil
}

func (p *gRPCProxy) getConnection() (*grpc.ClientConn, error) {
    p.connMu.RLock()
    conn := p.conn
    p.connMu.RUnlock()
    
    if conn != nil {
        return conn, nil
    }
    
    if err := p.ensureConnection(); err != nil {
        return nil, err
    }
    
    p.connMu.RLock()
    defer p.connMu.RUnlock()
    return p.conn, nil
}

func (h *Handler) handleGRPC(w http.ResponseWriter, r *http.Request) {
    if r.ProtoMajor != 2 {
        http.Error(w, "gRPC requires HTTP/2", http.StatusHTTPVersionNotSupported)
        return
    }
    
    ct := r.Header.Get("Content-Type")
    if !strings.HasPrefix(ct, "application/grpc") {
        http.Error(w, "invalid gRPC request content-type", http.StatusUnsupportedMediaType)
        return
    }
    
    if h.grpcProxy != nil && h.grpcProxy.director != nil {
        h.grpcProxy.director(r)
    }
    
    parts := strings.Split(r.URL.Path, "/")
    if len(parts) < 3 {
        http.Error(w, "invalid gRPC path", http.StatusBadRequest)
        return
    }
    
    fullMethod := r.URL.Path
    
    conn, err := h.grpcProxy.getConnection()
    if err != nil {
        writeGRPCError(w, status.Error(codes.Unavailable, "upstream unavailable"))
        return
    }
    
    ctx := r.Context()
    if h.route.Timeout != nil && h.route.Timeout.Read > 0 {
        var cancel context.CancelFunc
        ctx, cancel = context.WithTimeout(ctx, h.route.Timeout.Read)
        defer cancel()
    }
    
    md := extractMetadata(r.Header)
    ctx = metadata.NewOutgoingContext(ctx, md)
    
    if isStreamingRequest(r) {
        h.handleStreamingGRPC(ctx, w, r, conn, fullMethod)
    } else {
        h.handleUnaryGRPC(ctx, w, r, conn, fullMethod)
    }
}

func (h *Handler) handleUnaryGRPC(ctx context.Context, w http.ResponseWriter, r *http.Request, conn *grpc.ClientConn, method string) {
    reqBody, err := io.ReadAll(r.Body)
    if err != nil {
        writeGRPCError(w, status.Error(codes.InvalidArgument, "failed to read request"))
        return
    }
    defer r.Body.Close()
    
    codec := &rawCodec{}
    
    var respBody []byte
    var respHeader metadata.MD
    var respTrailer metadata.MD
    
    err = conn.Invoke(
        ctx,
        method,
        reqBody,
        &respBody,
        grpc.ForceCodec(codec),
        grpc.Header(&respHeader),
        grpc.Trailer(&respTrailer),
    )
    
    writeGRPCHeaders(w, respHeader)
    
    if err != nil {
        writeGRPCError(w, err)
        writeGRPCTrailers(w, respTrailer)
        return
    }
    
    w.WriteHeader(http.StatusOK)
    w.Write(respBody)
    
    writeGRPCTrailers(w, respTrailer)
}

func (h *Handler) handleStreamingGRPC(ctx context.Context, w http.ResponseWriter, r *http.Request, conn *grpc.ClientConn, method string) {
    flusher, ok := w.(http.Flusher)
    if !ok {
        writeGRPCError(w, status.Error(codes.Internal, "streaming not supported"))
        return
    }
    
    desc := &grpc.StreamDesc{
        StreamName:    method,
        ServerStreams: true,
        ClientStreams: true,
    }
    
    codec := &rawCodec{}
    
    stream, err := conn.NewStream(ctx, desc, method, grpc.ForceCodec(codec))
    if err != nil {
        writeGRPCError(w, err)
        return
    }
    defer stream.CloseSend()
    
    headers, err := stream.Header()
    if err != nil {
        writeGRPCError(w, err)
        return
    }
    
    writeGRPCHeaders(w, headers)
    w.WriteHeader(http.StatusOK)
    flusher.Flush()
    
    errChan := make(chan error, 2)
    
    go func() {
        defer close(errChan)
        
        for {
            frame, err := readGRPCFrame(r.Body)
            if err == io.EOF {
                stream.CloseSend()
                return
            }
            if err != nil {
                errChan <- err
                return
            }
            
            if err := stream.SendMsg(frame); err != nil {
                errChan <- err
                return
            }
        }
    }()
    
    go func() {
        for {
            var frame []byte
            err := stream.RecvMsg(&frame)
            if err == io.EOF {
                trailers := stream.Trailer()
                writeGRPCTrailers(w, trailers)
                flusher.Flush()
                errChan <- nil
                return
            }
            if err != nil {
                errChan <- err
                return
            }
            
            if err := writeGRPCFrame(w, frame); err != nil {
                errChan <- err
                return
            }
            flusher.Flush()
        }
    }()
    
    if err := <-errChan; err != nil {
        writeGRPCError(w, err)
    }
}

func extractMetadata(headers http.Header) metadata.MD {
    md := metadata.MD{}
    
    for key, values := range headers {
        key = strings.ToLower(key)
        
        if strings.HasPrefix(key, "grpc-") || key == "authorization" {
            md[key] = values
        }
        
        if strings.HasPrefix(key, "x-") {
            md[key] = values
        }
    }
    
    return md
}

func writeGRPCHeaders(w http.ResponseWriter, md metadata.MD) {
    for key, values := range md {
        key = http.CanonicalHeaderKey(key)
        for _, value := range values {
            w.Header().Add(key, value)
        }
    }
    w.Header().Set("Content-Type", "application/grpc")
}

func writeGRPCTrailers(w http.ResponseWriter, md metadata.MD) {
    for key, values := range md {
        key = "Trailer-" + http.CanonicalHeaderKey(key)
        for _, value := range values {
            w.Header().Add(key, value)
        }
    }
}

func writeGRPCError(w http.ResponseWriter, err error) {
    st, ok := status.FromError(err)
    if !ok {
        st = status.New(codes.Unknown, err.Error())
    }
    
    w.Header().Set("Content-Type", "application/grpc")
    w.Header().Set("Grpc-Status", fmt.Sprintf("%d", st.Code()))
    w.Header().Set("Grpc-Message", st.Message())
    
    w.WriteHeader(http.StatusOK)
}

func isStreamingRequest(r *http.Request) bool {
    te := r.Header.Get("TE")
    return strings.Contains(te, "trailers") || r.ContentLength == -1
}

func readGRPCFrame(r io.Reader) ([]byte, error) {
    header := make([]byte, 5)
    if _, err := io.ReadFull(r, header); err != nil {
        return nil, err
    }
    
    compressed := header[0] == 1
    length := binary.BigEndian.Uint32(header[1:5])
    
    if length > 16*1024*1024 {
        return nil, fmt.Errorf("message too large: %d bytes", length)
    }
    
    data := make([]byte, length)
    if _, err := io.ReadFull(r, data); err != nil {
        return nil, err
    }
    
    if compressed {
        return nil, fmt.Errorf("compressed messages not supported")
    }
    
    frame := make([]byte, 5+length)
    copy(frame, header)
    copy(frame[5:], data)
    
    return frame, nil
}

func writeGRPCFrame(w io.Writer, frame []byte) error {
    _, err := w.Write(frame)
    return err
}

type rawCodec struct{}

func (c *rawCodec) Marshal(v interface{}) ([]byte, error) {
    switch vv := v.(type) {
    case []byte:
        return vv, nil
    case *[]byte:
        return *vv, nil
    default:
        return nil, fmt.Errorf("unsupported type: %T", v)
    }
}

func (c *rawCodec) Unmarshal(data []byte, v interface{}) error {
    switch vv := v.(type) {
    case *[]byte:
        *vv = data
        return nil
    default:
        return fmt.Errorf("unsupported type: %T", v)
    }
}

func (c *rawCodec) Name() string {
    return "raw"
}

func (p *gRPCProxy) Close() error {
    p.connMu.Lock()
    defer p.connMu.Unlock()
    
    if p.conn != nil {
        err := p.conn.Close()
        p.conn = nil
        return err
    }
    
    return nil
}
