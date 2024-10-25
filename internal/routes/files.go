package routes

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type getFileRequest struct {
	FileName string `param:"file_name" validate:"required"`
}

func (h *Handler) GetFile(c echo.Context) error {
	req := new(getFileRequest)
	if err := bindAndValidate(c, req); err != nil {
		return err
	}
	presignedReq, err := h.BlobStore.PresignGetObject(c.Request().Context(), h.Config.S3BucketName, req.FileName)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, presignedReq)
}

type putFileRequest struct {
	File string `form:"file" validate:"required"`
}

func (h *Handler) PutFile(c echo.Context) error {
	req := new(putFileRequest)
	if err := bindAndValidate(c, req); err != nil {
		return err
	}
	file, err := c.FormFile("file")
	if err != nil {
		return err
	}
	f, err := file.Open()
	if err != nil {
		return err
	}
	defer f.Close()
	// fileContent, err := io.ReadAll(f)
	// err = h.fs.Upload(c.Request().Context(), h.config.S3BucketName, file.Filename, fileContent)
	// if err != nil {
	// 	return unexpectedError(c,err)
	// }
	return c.NoContent(http.StatusOK)
}

func (h *Handler) GetFileList(c echo.Context) error {
	files, err := h.BlobStore.GetList(c.Request().Context(), h.Config.S3BucketName, "")
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, files)
}
