package gui

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

//go:embed all:frontend
var frontendFS embed.FS

// NewHandler builds the HTTP handler backing the webview.
//
// The frontend talks to Go over plain HTTP rather than through generated Wails
// bindings: it keeps the frontend dependency-free (no npm, no binding
// generation step in the build) and, more importantly, lets the 280 KB - 1.6 MB
// SVG templates be streamed as ordinary responses instead of being marshalled
// through the JS bridge.
func NewHandler(state *State) http.Handler {
	mux := http.NewServeMux()

	registerSessionRoutes(mux, state)
	registerConfigRoutes(mux, state)
	registerDeviceRoutes(mux, state)
	registerExportRoutes(mux, state)
	registerDiagramRoutes(mux, state)
	registerTipsRoutes(mux, state)
	registerAboutRoutes(mux, state)

	// Templates and generated diagrams are served from disk by URL. Both roots
	// come from the configuration, so both need a traversal guard.
	mux.HandleFunc("GET /api/template", serveFileUnder(func() string {
		return state.Config().TemplatesDirectory
	}))
	mux.HandleFunc("GET /api/diagram", serveFileUnder(func() string {
		return state.Config().OutputDirectory
	}))

	mux.Handle("/", frontendHandler())

	return mux
}

// frontendHandler serves the user interface. It normally comes from the binary
// itself, but setting SIMDIAG_FRONTEND_DIR serves it from disk instead so the
// HTML/CSS/JS can be edited and reloaded without rebuilding (see make gui-dev).
func frontendHandler() http.Handler {
	if dir := os.Getenv("SIMDIAG_FRONTEND_DIR"); dir != "" {
		return withJavaScriptType(http.FileServer(http.Dir(dir)))
	}

	sub, err := fs.Sub(frontendFS, "frontend")
	if err != nil {
		panic(fmt.Sprintf("embedded frontend unavailable: %v", err))
	}
	return withJavaScriptType(http.FileServerFS(sub))
}

// withJavaScriptType states the type of every .js response instead of letting
// net/http guess it.
//
// The page is loaded as an ES module, and a module served with the wrong
// Content-Type is refused outright by the browser rather than tolerated the way
// a classic script would be. The guess comes from mime.TypeByExtension, which on
// Windows reads HKCR\.js\Content Type: a machine where some installer wrote
// text/plain there would open SimDiag on a blank window with nothing to explain
// it, and only that machine would see it.
func withJavaScriptType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".js") {
			w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		}
		next.ServeHTTP(w, r)
	})
}

// serveFileUnder serves the file named by the "p" query parameter, resolved
// against a root directory that is looked up per request (the user can change it
// while the app runs).
func serveFileUnder(root func() string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rel := r.URL.Query().Get("p")
		if rel == "" {
			http.Error(w, "missing p parameter", http.StatusBadRequest)
			return
		}

		path, err := safeJoin(root(), rel)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		http.ServeFile(w, r, path)
	}
}

// safeJoin resolves rel under root and refuses anything that escapes it.
// Without this, "p=../../../../Users/me/.ssh/id_rsa" would happily be served:
// the webview is a browser, and the page it renders is only as trustworthy as
// the SVG templates it loads.
func safeJoin(root, rel string) (string, error) {
	if root == "" {
		return "", fmt.Errorf("directory not configured")
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("invalid root directory")
	}

	// filepath.Join cleans the result, collapsing any ".." segments.
	target := filepath.Join(absRoot, filepath.FromSlash(rel))

	inside, err := filepath.Rel(absRoot, target)
	if err != nil || inside == ".." || strings.HasPrefix(inside, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path outside the allowed directory")
	}

	return target, nil
}

// existingDirectory returns the directory a picker should open at: the path
// itself when it is a directory, otherwise its parent, or "" if neither exists.
func existingDirectory(path string) string {
	if path == "" {
		return ""
	}
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		return path
	}
	parent := filepath.Dir(path)
	if info, err := os.Stat(parent); err == nil && info.IsDir() {
		return parent
	}
	return ""
}

// writeMessageError refuses a request with a message the interface can render in
// the user's language, rather than with a sentence in English.
//
// It is for the refusals the user is meant to act on: an export in flight, a
// missing configuration file. Genuinely technical failures (bad paths, I/O,
// malformed requests) keep using http.Error: they stay in English, and the
// sentence the frontend wraps them in is translated.
func writeMessageError(w http.ResponseWriter, status int, code string, args map[string]string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(msgArgs(code, args)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func writeJSON(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
