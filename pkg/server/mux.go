package server

import (
	"fmt"
	"net/http"
	"path"

	"github.com/charlieegan3/storage-console/pkg/database"
	"github.com/charlieegan3/storage-console/pkg/importer"
	metaRunner "github.com/charlieegan3/storage-console/pkg/meta/runner"
	propRunner "github.com/charlieegan3/storage-console/pkg/properties/runner"
	"github.com/charlieegan3/storage-console/pkg/server/handlers"
	"github.com/charlieegan3/storage-console/pkg/server/handlers/browse"
	"github.com/charlieegan3/storage-console/pkg/server/middlewares"
)

func newMux(opts *handlers.Options) (*http.ServeMux, error) {
	mux := http.NewServeMux()

	stylesEtag, stylesHandler, err := handlers.BuildCSSHandler(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to build styles handler: %s", err)
	}

	scriptETag, scriptHandler, err := handlers.BuildJSHandler(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to build script handler: %s", err)
	}

	opts.EtagStyles = stylesEtag
	opts.EtagScript = scriptETag

	indexHandler, err := handlers.BuildIndexHandler(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to build index handler: %s", err)
	}

	browseHandler, err := browse.BuildHandler(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to build browse handler: %s", err)
	}

	mux.Handle(
		"/reload",
		middlewares.BuildAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			opts.LoggerInfo.Printf("reloading")

			var prefix string
			if r.Method == http.MethodPost {
				err := r.ParseForm()
				if err == nil {
					prefix = r.FormValue("prefix")
				}
				if prefix != "" {
					opts.LoggerInfo.Printf("prefix: %q", prefix)

					txn, err := database.NewTxnWithSchema(opts.DB, "storage_console")
					if err != nil {
						opts.LoggerError.Printf("error creating transaction: %v", err)

						return
					}

					deleteMetadataStateSQL := `
with blob_ids as (
    select blobs.id
    from blobs
    join object_blobs on object_blobs.blob_id = blobs.id
    join objects on object_blobs.object_id = objects.id
    where objects.key = $1
)
delete from blob_metadata
where blob_id in (select id from blob_ids);
`

					_, err = txn.Exec(deleteMetadataStateSQL, prefix)
					if err != nil {
						opts.LoggerError.Printf("error cleaning state: %v", err)

						return
					}

					deletePropertiesStateSQL := `
with blob_ids as (
    select blobs.id
    from blobs
    join object_blobs on object_blobs.blob_id = blobs.id
    join objects on object_blobs.object_id = objects.id
    where objects.key = $1
)
delete from blob_properties
where blob_id in (select id from blob_ids);
`

					_, err = txn.Exec(deletePropertiesStateSQL, prefix)
					if err != nil {
						opts.LoggerError.Printf("error cleaning state: %v", err)

						return
					}

					err = txn.Commit()
					if err != nil {
						opts.LoggerError.Printf("error committing transaction: %v", err)
						return
					}
				}
			}

			_, err := importer.Run(r.Context(), opts.DB, opts.S3, &importer.Options{
				BucketName:  opts.BucketName,
				SchemaName:  "storage_console",
				Prefix:      prefix,
				LoggerInfo:  opts.LoggerInfo,
				LoggerError: opts.LoggerError,
			})
			if err != nil {
				opts.LoggerError.Printf("error running importer: %v", err)
				return
			}

			// do initial metadata processing
			_, err = metaRunner.Run(r.Context(), opts.DB, opts.S3, &metaRunner.Options{
				BucketName:        opts.BucketName,
				SchemaName:        "storage_console",
				Prefix:            prefix,
				EnabledProcessors: []string{"thumbnail", "exif", "color"},
				LoggerInfo:        opts.LoggerInfo,
				LoggerError:       opts.LoggerError,
			})
			if err != nil {
				opts.LoggerError.Printf("error running metadata runner: %v", err)
				return
			}

			// upgrade metadata into rich properties
			_, err = propRunner.Run(r.Context(), opts.DB, opts.S3, &propRunner.Options{
				BucketName:        opts.BucketName,
				SchemaName:        "storage_console",
				Prefix:            prefix,
				EnabledProcessors: []string{"exif", "color"},
				LoggerInfo:        opts.LoggerInfo,
				LoggerError:       opts.LoggerError,
			})
			if err != nil {
				opts.LoggerError.Printf("error running properties runner: %v", err)

				return
			}

			opts.LoggerInfo.Printf("reloaded")

			if prefix != "" {
				file := path.Base(prefix)
				dir := path.Dir(prefix)
				http.Redirect(w, r, "/b/"+dir+"/?preview="+file, http.StatusSeeOther)
				return
			}

			http.Redirect(w, r, "/", http.StatusSeeOther)
		}), opts),
	)

	mux.Handle(
		"/b/",
		middlewares.BuildAuth(http.HandlerFunc(browseHandler), opts),
	)

	mux.Handle(
		"/icons/content-types/",
		middlewares.BuildAuth(http.HandlerFunc(handlers.BuildContentTypeIconHandler(opts)), opts),
	)

	mux.Handle(
		"/script.js",
		middlewares.BuildAuth(http.HandlerFunc(scriptHandler), opts),
	)
	mux.Handle(
		"/styles.css",
		middlewares.BuildAuth(http.HandlerFunc(stylesHandler), opts),
	)
	mux.Handle(
		"/",
		middlewares.BuildAuth(http.HandlerFunc(indexHandler), opts),
	)

	return mux, nil
}
