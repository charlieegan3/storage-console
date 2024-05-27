package browse

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/minio/minio-go/v7"

	"github.com/charlieegan3/storage-console/pkg/server/handlers"
)

type browseEntry struct {
	Name        string
	Key         string
	IsDir       bool
	ContentType string
	Size        string
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

	mc, ok := opts.ObjectStorageProviders["local-minio"]
	if !ok {
		return nil, fmt.Errorf("local-minio object storage provider is required")
	}

	tmpl, err := template.ParseFS(
		handlers.Templates,
		"templates/browse.html",
		"templates/base.html",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %s", err)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/") {
			renderDir(opts, mc, tmpl)(w, r)
			return
		} else {
			_, err := mc.StatObject(
				r.Context(),
				"local",
				strings.TrimPrefix(r.URL.Path, "/b/"),
				minio.StatObjectOptions{},
			)
			if err == nil {
				renderFile(opts, mc)(w, r)
				return
			}
		}

		w.WriteHeader(http.StatusBadRequest)
		_, err := w.Write([]byte("unknown path type"))
		if err != nil && opts.LoggerError != nil {
			opts.LoggerError.Println(err)
		}
	}, nil
}

func renderFile(opts *handlers.Options, mc *minio.Client) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		obj, err := mc.GetObject(
			r.Context(),
			"local",
			strings.TrimPrefix(r.URL.Path, "/b/"),
			minio.GetObjectOptions{},
		)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)

			_, err = w.Write([]byte(err.Error()))
			if err != nil && opts.LoggerError != nil {
				opts.LoggerError.Println(err)
			}

			return
		}

		metadata, err := obj.Stat()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)

			_, err = w.Write([]byte(err.Error()))
			if err != nil && opts.LoggerError != nil {
				opts.LoggerError.Println(err)
			}

			return
		}

		w.Header().Set("Content-Type", metadata.ContentType)
		w.Header().Set("Content-Length", fmt.Sprintf("%d", metadata.Size))

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

func renderDir(opts *handlers.Options, mc *minio.Client, tmpl *template.Template) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/b/")
		// a trailing / is required for path prefix listing,
		// unless we are listing the root
		if !strings.HasSuffix(path, "/") && path != "" {
			path = path + "/"
		}

		bcs := breadcrumbsFromPath(path)

		var keys []interface{}
		var dirSizeArgs []interface{}
		var orderedKeys []string
		entries := make(map[string]*browseEntry)
		for obj := range mc.ListObjects(
			r.Context(),
			"local",
			minio.ListObjectsOptions{
				Prefix:    path,
				Recursive: false,
			},
		) {
			orderedKeys = append(orderedKeys, obj.Key)
			isDir := strings.HasSuffix(obj.Key, "/")

			contentType := "custom/unknown"
			if isDir {
				contentType = "custom/folder"
			}

			entries[obj.Key] = &browseEntry{
				Name:        filepath.Base(obj.Key),
				Key:         obj.Key,
				IsDir:       isDir,
				ContentType: contentType,
			}

			if !isDir {
				keys = append(keys, obj.Key)
			} else {
				dirSizeArgs = append(dirSizeArgs, obj.Key)
			}
		}

		var placeholders string
		for i := range keys {
			placeholders += fmt.Sprintf("$%d", i+1)
			if i < len(keys)-1 {
				placeholders += ", "
			}
		}
		loadMetadataSQL := fmt.Sprintf(`
SELECT key, size, md5, content_types.name AS content_type FROM objects
LEFT JOIN object_blobs ON object_blobs.object_id = objects.id
LEFT JOIN blobs ON object_blobs.blob_id = blobs.id
LEFT JOIN content_types ON blobs.content_type_id = content_types.id
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
			err = rows.Scan(&key, &size, &md5, &contentType)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)

				_, err = w.Write([]byte(err.Error()))
				if err != nil && opts.LoggerError != nil {
					opts.LoggerError.Println(err)
				}

				return
			}

			if e, ok := entries[key]; ok {
				e.ContentType = contentType
				e.Size = humanizeBytes(size)
			}
		}

		var entryList []*browseEntry
		for _, key := range orderedKeys {
			entryList = append(entryList, entries[key])
		}

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

		err = tmpl.ExecuteTemplate(buf, "base", struct {
			Opts        *handlers.Options
			Entries     []*browseEntry
			Breadcrumbs breadcrumbs
		}{
			Opts:        opts,
			Entries:     entryList,
			Breadcrumbs: bcs,
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
