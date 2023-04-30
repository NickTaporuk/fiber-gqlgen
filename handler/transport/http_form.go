package transport

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"mime"
	"os"
	"strings"

	"github.com/99designs/gqlgen/graphql"
	fibergqlgen "github.com/NickTaporuk/fiber-gqlgen"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/utils"
)

// MultipartForm the Multipart request spec https://github.com/jaydenseric/graphql-multipart-request-spec
type MultipartForm struct {
	// MaxUploadSize sets the maximum number of bytes used to parse a request body
	// as multipart/form-data.
	MaxUploadSize int64

	// MaxMemory defines the maximum number of bytes used to parse a request body
	// as multipart/form-data in memory, with the remainder stored on disk in
	// temporary files.
	MaxMemory int64
}

var _ fibergqlgen.Transport = MultipartForm{}

func (f MultipartForm) Supports(c *fiber.Ctx) bool {
	if c.Get("Upgrade") != "" {
		return false
	}

	mediaType, _, err := mime.ParseMediaType(c.Get("Content-Type"))
	if err != nil {
		return false
	}

	return c.Method() == "POST" && mediaType == "multipart/form-data"
}

func (f MultipartForm) Do(c *fiber.Ctx, exec graphql.GraphExecutor) error {
	c.Set("Content-Type", "application/json")

	start := graphql.Now()

	var err error
	if int64(c.Request().Header.ContentLength()) > f.maxUploadSize() {
		return writeJsonError(c, "failed to parse multipart form, request body too large")
	}
	if _, err = c.MultipartForm(); err != nil {
		c.Status(fiber.StatusUnprocessableEntity)
		if strings.Contains(err.Error(), "request body too large") {
			return writeJsonError(c, "failed to parse multipart form, request body too large")
		}
		return writeJsonError(c, "failed to parse multipart form")
	}

	var params graphql.RawParams

	if err = c.App().Config().JSONDecoder(utils.UnsafeBytes(c.FormValue("operations")), &params); err != nil {
		c.Status(fiber.StatusUnprocessableEntity)
		return writeJsonError(c, "operations form field could not be decoded")
	}

	uploadsMap := map[string][]string{}
	if err = json.Unmarshal([]byte(c.FormValue("map")), &uploadsMap); err != nil {
		c.Status(fiber.StatusUnprocessableEntity)
		return writeJsonError(c, "map form field could not be decoded")
	}

	var upload graphql.Upload
	for key, paths := range uploadsMap {
		if len(paths) == 0 {
			c.Status(fiber.StatusUnprocessableEntity)
			return writeJsonErrorf(c, "invalid empty operations paths list for key %s", key)
		}
		header, err := c.FormFile(key)
		if err != nil {
			c.Status(fiber.StatusUnprocessableEntity)
			return writeJsonErrorf(c, "failed to get key %s from form", key)
		}

		file, err := header.Open()
		if err != nil {
			return err
		}
		defer file.Close()

		if len(paths) == 1 {
			upload = graphql.Upload{
				File:        file,
				Size:        header.Size,
				Filename:    header.Filename,
				ContentType: header.Header.Get("Content-Type"),
			}

			if err := params.AddUpload(upload, key, paths[0]); err != nil {
				c.Status(fiber.StatusUnprocessableEntity)
				return writeJsonGraphqlError(c, err)
			}
		} else {
			if int64(c.Request().Header.ContentLength()) < f.maxMemory() {
				fileBytes, err := ioutil.ReadAll(file)
				if err != nil {
					c.Status(fiber.StatusUnprocessableEntity)
					return writeJsonErrorf(c, "failed to read file for key %s", key)
				}
				for _, path := range paths {
					upload = graphql.Upload{
						File:        &bytesReader{s: &fileBytes, i: 0, prevRune: -1},
						Size:        header.Size,
						Filename:    header.Filename,
						ContentType: header.Header.Get("Content-Type"),
					}

					if err := params.AddUpload(upload, key, path); err != nil {
						c.Status(fiber.StatusUnprocessableEntity)
						return writeJsonGraphqlError(c, err)
					}
				}
			} else {
				tmpFile, err := ioutil.TempFile(os.TempDir(), "gqlgen-")
				if err != nil {
					c.Status(fiber.StatusUnprocessableEntity)
					return writeJsonErrorf(c, "failed to create temp file for key %s", key)
				}
				tmpName := tmpFile.Name()
				defer func() {
					_ = os.Remove(tmpName)
				}()
				_, err = io.Copy(tmpFile, file)
				if err != nil {
					c.Status(fiber.StatusUnprocessableEntity)
					if err := tmpFile.Close(); err != nil {
						return writeJsonErrorf(c, "failed to copy to temp file and close temp file for key %s", key)
					}
					return writeJsonErrorf(c, "failed to copy to temp file for key %s", key)
				}
				if err := tmpFile.Close(); err != nil {
					c.Status(fiber.StatusUnprocessableEntity)
					return writeJsonErrorf(c, "failed to close temp file for key %s", key)
				}
				for _, path := range paths {
					pathTmpFile, err := os.Open(tmpName)
					if err != nil {
						c.Status(fiber.StatusUnprocessableEntity)
						return writeJsonErrorf(c, "failed to open temp file for key %s", key)
					}
					defer pathTmpFile.Close()
					upload = graphql.Upload{
						File:        pathTmpFile,
						Size:        header.Size,
						Filename:    header.Filename,
						ContentType: header.Header.Get("Content-Type"),
					}

					if err := params.AddUpload(upload, key, path); err != nil {
						c.Status(fiber.StatusUnprocessableEntity)
						return writeJsonGraphqlError(c, err)
					}
				}
			}
		}
	}

	params.ReadTime = graphql.TraceTiming{
		Start: start,
		End:   graphql.Now(),
	}

	rc, gerr := exec.CreateOperationContext(c.Context(), &params)
	if gerr != nil {
		resp := exec.DispatchError(graphql.WithOperationContext(c.Context(), rc), gerr)
		c.Status(statusFor(gerr))
		return writeJson(c, resp)
	}
	responses, ctx := exec.DispatchOperation(c.Context(), rc)
	return writeJson(c, responses(ctx))
}

func (f MultipartForm) maxUploadSize() int64 {
	if f.MaxUploadSize == 0 {
		return 32 << 20
	}
	return f.MaxUploadSize
}

func (f MultipartForm) maxMemory() int64 {
	if f.MaxMemory == 0 {
		return 32 << 20
	}
	return f.MaxMemory
}
