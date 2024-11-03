package browse

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"

	"github.com/charlieegan3/storage-console/pkg/server/handlers"
)

const (
	dataPath = "data/"
	metaPath = "meta/"
)

type browseEntry struct {
	Name        string
	Key         string
	IsDir       bool
	ContentType string
	Size        string
	HasThumb    bool
	MD5         string
}

type breadcrumbs struct {
	Display bool
	Items   []breadcrumb
}

type breadcrumb struct {
	Name      string
	Path      string
	Navigable bool
}

func BuildHandler(opts *handlers.Options) (func(http.ResponseWriter, *http.Request), error) {
	if opts.DB == nil {
		return nil, fmt.Errorf("DB is required")
	}

	mc := opts.S3

	tmplDir, err := template.ParseFS(
		handlers.Templates,
		"templates/browse.html",
		"templates/base.html",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to parse dir templates: %s", err)
	}

	tmplDirGrid, err := template.New("grid").Funcs(template.FuncMap{"join": func(sep string, s ...string) string {
		return strings.Join(s, sep)
	}}).ParseFS(
		handlers.Templates,
		"templates/browse-grid.html",
		"templates/base.html",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to parse dir grid templates: %s", err)
	}

	tmplFile, err := template.ParseFS(
		handlers.Templates,
		"templates/browse-preview.html",
		"templates/base.html",
	)

	return func(w http.ResponseWriter, r *http.Request) {
		preview := r.URL.Query().Get("preview")
		asset := r.URL.Query().Get("asset")
		thumb := r.URL.Query().Get("thumb")
		download := r.URL.Query().Get("download")
		view := r.URL.Query().Get("view")

		// then render the object
		if asset != "" {
			objectPath := strings.TrimPrefix(r.URL.Path+asset, "/b/")

			if thumb != "" {
				objectPath := strings.TrimPrefix(r.URL.Path+asset, "/b/")

				renderObject(opts, mc, objectPath, download != "", thumb)(w, r)
				return
			}

			renderObject(opts, mc, objectPath, download != "", "")(w, r)

			return
		}

		// then render the file
		if preview != "" {
			objectPath := path.Join("data", strings.TrimPrefix(r.URL.Path+preview, "/b/"))

			renderPreview(opts, mc, tmplFile, objectPath)(w, r)
			return
		}

		// render the directory
		if strings.HasSuffix(r.URL.Path, "/") {
			if view == "grid" {
				renderDir(opts, mc, tmplDirGrid)(w, r)

				return
			}

			renderDir(opts, mc, tmplDir)(w, r)

			return
		}

		w.WriteHeader(http.StatusBadRequest)

		_, err = w.Write([]byte("unknown path type"))
		if err != nil && opts.LoggerError != nil {
			opts.LoggerError.Println(err)
		}
	}, nil
}

func renderObject(
	opts *handlers.Options,
	mc *minio.Client,
	objectPath string,
	download bool,
	thumbKey string,
) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var obj *minio.Object
		var err error
		var p string

		if thumbKey != "" {
			p = path.Join(metaPath, fmt.Sprintf("thumbnail/%s.jpg", thumbKey))
		} else {
			p = filepath.Join(dataPath, objectPath)
		}

		obj, err = mc.GetObject(
			r.Context(),
			opts.BucketName,
			p,
			minio.StatObjectOptions{},
		)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)

			_, err = w.Write([]byte(err.Error()))
			if err != nil && opts.LoggerError != nil {
				opts.LoggerError.Println(err)
			}

			return
		}

		stat, err := obj.Stat()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, err = w.Write([]byte(err.Error()))
			if err != nil && opts.LoggerError != nil {
				opts.LoggerError.Println(err)
			}

			return
		}

		w.Header().Set("Content-Type", stat.ContentType)
		w.Header().Set("Content-Length", fmt.Sprintf("%d", stat.Size))
		w.Header().Set("ETag", stat.ETag)
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		w.Header().Set("Expires", time.Now().AddDate(10, 0, 0).Format(http.TimeFormat))
		if download {
			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filepath.Base(objectPath)))
		} else {
			w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", filepath.Base(objectPath)))
		}

		if r.Header.Get("If-None-Match") == stat.ETag {
			w.WriteHeader(http.StatusNotModified)

			return
		}

		_, err = io.Copy(w, obj)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)

			_, err = w.Write([]byte(err.Error()))
			if err != nil && opts.LoggerError != nil {
				opts.LoggerError.Println(err)
			}

			return
		}
	}
}

func renderPreview(
	opts *handlers.Options,
	mc *minio.Client,
	tmpl *template.Template,
	objectPath string,
) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		_, err := mc.StatObject(
			r.Context(),
			opts.BucketName,
			objectPath,
			minio.StatObjectOptions{},
		)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)

			_, err = w.Write([]byte(err.Error()))
			if err != nil && opts.LoggerError != nil {
				opts.LoggerError.Println(err)
			}

			return
		}

		buf := bytes.NewBuffer([]byte{})

		viewPath := strings.TrimPrefix(objectPath, dataPath)

		blobDetailsSQL := `
select blobs.size, last_modified, md5, content_types.name from objects
left join object_blobs on objects.id = object_blobs.object_id
left join blobs on blobs.id = object_blobs.blob_id
left join content_types on blobs.content_type_id = content_types.id
where key = $1`
		var size int64
		var lastModified time.Time
		var md5, contentType string
		err = opts.DB.QueryRow(
			blobDetailsSQL,
			viewPath,
		).Scan(&size, &lastModified, &md5, &contentType)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)

			_, err = w.Write([]byte(err.Error()))
			if err != nil && opts.LoggerError != nil {
				opts.LoggerError.Println(err)
			}

			return
		}

		previewableContentTypes := []string{
			"image/png",
			"image/jpeg",
		}

		dir := filepath.Dir(viewPath)

		if dir == "." {
			dir = "/"
		}

		err = tmpl.ExecuteTemplate(buf, "base", struct {
			Opts                   *handlers.Options
			Breadcrumbs            breadcrumbs
			ContentType            string
			ContentTypePreviewable bool
			Dir                    string
			File                   string
			LastModified           string
			MD5                    string
			Size                   string
		}{
			Opts:                   opts,
			Breadcrumbs:            breadcrumbsFromPath(viewPath),
			ContentType:            contentType,
			ContentTypePreviewable: slices.Contains(previewableContentTypes, contentType),
			Dir:                    dir,
			File:                   filepath.Base(objectPath),
			LastModified:           lastModified.Format(time.RFC3339),
			MD5:                    md5,
			Size:                   humanizeBytes(size),
		})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)

			_, err = w.Write([]byte(err.Error()))
			if err != nil && opts.LoggerError != nil {
				opts.LoggerError.Println(err)
			}

			return
		}

		_, err = io.Copy(w, buf)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)

			_, err = w.Write([]byte(err.Error()))
			if err != nil && opts.LoggerError != nil {
				opts.LoggerError.Println(err)
			}

			return
		}
	}
}

func renderDir(opts *handlers.Options, mc *minio.Client, tmpl *template.Template) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		viewPath := strings.TrimPrefix(r.URL.Path, "/b/")

		p := path.Join(dataPath, viewPath)
		// a trailing / is required for path prefix listing,
		// unless we are listing the root
		if !strings.HasSuffix(p, "/") && p != "" {
			p = p + "/"
		}

		var keys []interface{}
		var dirSizeArgs []interface{}
		var orderedKeys []string

		entries := make(map[string]*browseEntry)

		for obj := range mc.ListObjects(
			r.Context(),
			opts.BucketName,
			minio.ListObjectsOptions{
				Prefix:    p,
				Recursive: false,
			},
		) {
			key := strings.TrimPrefix(obj.Key, dataPath)

			orderedKeys = append(orderedKeys, key)
			isDir := strings.HasSuffix(key, "/")

			contentType := "custom/unknown"
			if isDir {
				contentType = "custom/folder"
			}

			entries[key] = &browseEntry{
				Name:        filepath.Base(key),
				Key:         key,
				IsDir:       isDir,
				ContentType: contentType,
			}

			if !isDir {
				keys = append(keys, key)
			} else {
				dirSizeArgs = append(dirSizeArgs, key)
			}
		}

		if len(keys) > 0 {
			var placeholders string
			for i := range keys {
				placeholders += fmt.Sprintf("$%d", i+1)
				if i < len(keys)-1 {
					placeholders += ", "
				}
			}

			loadMetadataSQL := fmt.Sprintf(`
SELECT
	key,
	size,
	md5,
	content_types.name AS content_type,
	COALESCE(blob_metadata.thumbnail, FALSE) as has_thumb
FROM objects
LEFT JOIN object_blobs ON object_blobs.object_id = objects.id
LEFT JOIN blobs ON object_blobs.blob_id = blobs.id
LEFT JOIN content_types ON blobs.content_type_id = content_types.id
LEFT JOIN blob_metadata ON blobs.id = blob_metadata.blob_id
WHERE key IN (%s)`, placeholders)

			rows, err := opts.DB.Query(loadMetadataSQL, keys...)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)

				_, err = w.Write([]byte(err.Error()))
				if err != nil && opts.LoggerError != nil {
					opts.LoggerError.Println(err)
				}

				return
			}

			for rows.Next() {
				var key string
				var size int64
				var md5 string
				var contentType string
				var hasThumb bool

				err = rows.Scan(&key, &size, &md5, &contentType, &hasThumb)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)

					_, err = w.Write([]byte(err.Error()))
					if err != nil && opts.LoggerError != nil {
						opts.LoggerError.Println(err)
					}

					return
				}

				if e, ok := entries[key]; ok {
					e.MD5 = md5
					e.ContentType = contentType
					e.Size = humanizeBytes(size)
					e.HasThumb = hasThumb
				}
			}
		}

		var entryList []*browseEntry
		for _, key := range orderedKeys {
			entryList = append(entryList, entries[key])
		}

		// calc the size of the directories based on the objects they contain
		if len(dirSizeArgs) > 0 {
			var sb strings.Builder
			for i := range dirSizeArgs {
				sb.WriteString(fmt.Sprintf("WHEN key ILIKE $%d || '%%' THEN $%d\n", i+1, i+1))
			}

			dirSizeSQL := fmt.Sprintf(`
select
    CASE
    	%s
        ELSE ''
    END AS dir,
    sum(size) from objects
left join object_blobs ON object_blobs.object_id = objects.id
left join blobs ON object_blobs.blob_id = blobs.id
left join content_types ON blobs.content_type_id = content_types.id
group by dir`, sb.String())

			dirSizeRows, err := opts.DB.Query(dirSizeSQL, dirSizeArgs...)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)

				_, err = w.Write([]byte(err.Error()))
				if err != nil && opts.LoggerError != nil {
					opts.LoggerError.Println(err)
				}

				return
			}

			for dirSizeRows.Next() {
				var dir string
				var size int64
				err = dirSizeRows.Scan(&dir, &size)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)

					_, err = w.Write([]byte(err.Error()))
					if err != nil && opts.LoggerError != nil {
						opts.LoggerError.Println(err)
					}

					return
				}

				if e, ok := entries[dir]; ok {
					e.Size = humanizeBytes(size)
				}
			}
		}

		buf := bytes.NewBuffer([]byte{})

		err := tmpl.ExecuteTemplate(buf, "base", struct {
			Opts        *handlers.Options
			Path        string
			Entries     []*browseEntry
			Breadcrumbs breadcrumbs
		}{
			Opts:        opts,
			Path:        r.URL.Path,
			Entries:     entryList,
			Breadcrumbs: breadcrumbsFromPath(viewPath),
		})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)

			_, err = w.Write([]byte(err.Error()))
			if err != nil && opts.LoggerError != nil {
				opts.LoggerError.Println(err)
			}

			return
		}

		_, err = io.Copy(w, buf)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)

			_, err = w.Write([]byte(err.Error()))
			if err != nil && opts.LoggerError != nil {
				opts.LoggerError.Println(err)
			}

			return
		}
	}
}
