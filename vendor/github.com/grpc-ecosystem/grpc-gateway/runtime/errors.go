package runtime

import (
<<<<<<< 130c674ed2ee159bf86e770605d1b6c1f5bc6f64
	"context"
=======
>>>>>>> Govendor update
	"io"
	"net/http"

	"github.com/golang/protobuf/proto"
<<<<<<< 130c674ed2ee159bf86e770605d1b6c1f5bc6f64
	"github.com/golang/protobuf/ptypes/any"
=======
	"golang.org/x/net/context"
>>>>>>> Govendor update
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/status"
)

// HTTPStatusFromCode converts a gRPC error code into the corresponding HTTP response status.
<<<<<<< 130c674ed2ee159bf86e770605d1b6c1f5bc6f64
// See: https://github.com/googleapis/googleapis/blob/master/google/rpc/code.proto
=======
>>>>>>> Govendor update
func HTTPStatusFromCode(code codes.Code) int {
	switch code {
	case codes.OK:
		return http.StatusOK
	case codes.Canceled:
		return http.StatusRequestTimeout
	case codes.Unknown:
		return http.StatusInternalServerError
	case codes.InvalidArgument:
		return http.StatusBadRequest
	case codes.DeadlineExceeded:
<<<<<<< 130c674ed2ee159bf86e770605d1b6c1f5bc6f64
		return http.StatusGatewayTimeout
=======
		return http.StatusRequestTimeout
>>>>>>> Govendor update
	case codes.NotFound:
		return http.StatusNotFound
	case codes.AlreadyExists:
		return http.StatusConflict
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	case codes.ResourceExhausted:
<<<<<<< 130c674ed2ee159bf86e770605d1b6c1f5bc6f64
		return http.StatusTooManyRequests
=======
		return http.StatusForbidden
>>>>>>> Govendor update
	case codes.FailedPrecondition:
		return http.StatusPreconditionFailed
	case codes.Aborted:
		return http.StatusConflict
	case codes.OutOfRange:
		return http.StatusBadRequest
	case codes.Unimplemented:
		return http.StatusNotImplemented
	case codes.Internal:
		return http.StatusInternalServerError
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	case codes.DataLoss:
		return http.StatusInternalServerError
	}

<<<<<<< 130c674ed2ee159bf86e770605d1b6c1f5bc6f64
	grpclog.Infof("Unknown gRPC error code: %v", code)
=======
	grpclog.Printf("Unknown gRPC error code: %v", code)
>>>>>>> Govendor update
	return http.StatusInternalServerError
}

var (
	// HTTPError replies to the request with the error.
	// You can set a custom function to this variable to customize error format.
	HTTPError = DefaultHTTPError
	// OtherErrorHandler handles the following error used by the gateway: StatusMethodNotAllowed StatusNotFound and StatusBadRequest
	OtherErrorHandler = DefaultOtherErrorHandler
)

type errorBody struct {
<<<<<<< 130c674ed2ee159bf86e770605d1b6c1f5bc6f64
	Error   string     `protobuf:"bytes,1,name=error" json:"error"`
	Code    int32      `protobuf:"varint,2,name=code" json:"code"`
	Details []*any.Any `protobuf:"bytes,3,rep,name=details" json:"details,omitempty"`
}

// Make this also conform to proto.Message for builtin JSONPb Marshaler
=======
	Error string `protobuf:"bytes,1,name=error" json:"error"`
	Code  int32  `protobuf:"varint,2,name=code" json:"code"`
}

//Make this also conform to proto.Message for builtin JSONPb Marshaler
>>>>>>> Govendor update
func (e *errorBody) Reset()         { *e = errorBody{} }
func (e *errorBody) String() string { return proto.CompactTextString(e) }
func (*errorBody) ProtoMessage()    {}

// DefaultHTTPError is the default implementation of HTTPError.
// If "err" is an error from gRPC system, the function replies with the status code mapped by HTTPStatusFromCode.
// If otherwise, it replies with http.StatusInternalServerError.
//
// The response body returned by this function is a JSON object,
// which contains a member whose key is "error" and whose value is err.Error().
func DefaultHTTPError(ctx context.Context, mux *ServeMux, marshaler Marshaler, w http.ResponseWriter, _ *http.Request, err error) {
	const fallback = `{"error": "failed to marshal error message"}`

	w.Header().Del("Trailer")
	w.Header().Set("Content-Type", marshaler.ContentType())

	s, ok := status.FromError(err)
	if !ok {
		s = status.New(codes.Unknown, err.Error())
	}

	body := &errorBody{
<<<<<<< 130c674ed2ee159bf86e770605d1b6c1f5bc6f64
		Error:   s.Message(),
		Code:    int32(s.Code()),
		Details: s.Proto().GetDetails(),
=======
		Error: s.Message(),
		Code:  int32(s.Code()),
>>>>>>> Govendor update
	}

	buf, merr := marshaler.Marshal(body)
	if merr != nil {
<<<<<<< 130c674ed2ee159bf86e770605d1b6c1f5bc6f64
		grpclog.Infof("Failed to marshal error message %q: %v", body, merr)
		w.WriteHeader(http.StatusInternalServerError)
		if _, err := io.WriteString(w, fallback); err != nil {
			grpclog.Infof("Failed to write response: %v", err)
=======
		grpclog.Printf("Failed to marshal error message %q: %v", body, merr)
		w.WriteHeader(http.StatusInternalServerError)
		if _, err := io.WriteString(w, fallback); err != nil {
			grpclog.Printf("Failed to write response: %v", err)
>>>>>>> Govendor update
		}
		return
	}

	md, ok := ServerMetadataFromContext(ctx)
	if !ok {
<<<<<<< 130c674ed2ee159bf86e770605d1b6c1f5bc6f64
		grpclog.Infof("Failed to extract ServerMetadata from context")
=======
		grpclog.Printf("Failed to extract ServerMetadata from context")
>>>>>>> Govendor update
	}

	handleForwardResponseServerMetadata(w, mux, md)
	handleForwardResponseTrailerHeader(w, md)
	st := HTTPStatusFromCode(s.Code())
	w.WriteHeader(st)
	if _, err := w.Write(buf); err != nil {
<<<<<<< 130c674ed2ee159bf86e770605d1b6c1f5bc6f64
		grpclog.Infof("Failed to write response: %v", err)
=======
		grpclog.Printf("Failed to write response: %v", err)
>>>>>>> Govendor update
	}

	handleForwardResponseTrailer(w, md)
}

// DefaultOtherErrorHandler is the default implementation of OtherErrorHandler.
// It simply writes a string representation of the given error into "w".
func DefaultOtherErrorHandler(w http.ResponseWriter, _ *http.Request, msg string, code int) {
	http.Error(w, msg, code)
}
