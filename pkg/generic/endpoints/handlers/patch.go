package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	jsonpatch "github.com/evanphx/json-patch"
	"github.com/gin-gonic/gin"
	"io"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"
	"lightiot/pkg/apis"
	"lightiot/pkg/apis/response"
	"lightiot/pkg/generic/meta"
	"lightiot/pkg/generic/rest"
	"net/http"
	"os"
	"strings"
)

const (
	maxJSONPatchOperations = 1000
)

func Patch(p rest.Patcher, patchTypes sets.String) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer c.Request.Body.Close()

		contentType := c.GetHeader("Content-Type")
		// Remove "; charset=" if included in header.
		if idx := strings.Index(contentType, ";"); idx > 0 {
			contentType = contentType[:idx]
		}

		if !patchTypes.Has(contentType) {
			c.Status(http.StatusUnsupportedMediaType)
			return
		}

		eTag := c.GetHeader(apis.IfMatch)
		if len(eTag) == 0 {
			c.Status(http.StatusPreconditionRequired)
			return
		}

		pathBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			klog.V(3).InfoS("Failed to read", "err", err)
			c.Status(http.StatusInternalServerError)
			return
		}

		id := c.Param("id")
		ctx := c.Request.Context()
		old, err := p.Get(ctx, id, nil)
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}

		versionedJS, err := json.Marshal(old)
		if err != nil {
			klog.V(3).InfoS("Failed to marshal", "err", err)
			c.Status(http.StatusInternalServerError)
			return
		}

		patchedJS, err := applyJSPatch(types.PatchType(contentType), pathBytes, versionedJS)
		if err != nil {
			c.JSONP(http.StatusBadRequest, response.NewMultiError(err))
			return
		}

		obj := p.New()
		if err := json.NewDecoder(bytes.NewBuffer(patchedJS)).Decode(obj); err != nil {
			klog.V(3).InfoS("Failed to decode", "err", err)
			c.JSON(http.StatusBadRequest, response.NewMultiError(response.ErrMalformedJSON))
			return
		}

		updated, err := p.Update(ctx, id, obj, rest.ValidateUpdateFunc(getValidateUpdate(p)), &meta.UpdateOptions{Version: eTag, Query: c.Request.URL.Query()})
		if err != nil {
			switch {
			case os.IsNotExist(err):
				c.Status(http.StatusNotFound)
			case errors.Is(err, apis.ErrMismatch):
				c.Status(http.StatusPreconditionFailed)
			default:
				if response.IsResponseError(err) {
					c.JSON(http.StatusBadRequest, response.NewMultiError(err))
				} else {
					c.Status(http.StatusInternalServerError)
				}
			}
			return
		}

		accessor, _ := meta.Accessor(updated)
		c.Header(apis.ETag, accessor.GetVersion())
		c.JSON(http.StatusOK, updated)
	}
}

func applyJSPatch(patchType types.PatchType, patchBytes, versionedJS []byte) (patchedJS []byte, err error) {
	switch patchType {
	case types.JSONPatchType:
		patchObj, err := jsonpatch.DecodePatch(patchBytes)
		if err != nil {
			return nil, response.ErrMalformedJSON
		}
		if len(patchObj) > maxJSONPatchOperations {
			klog.V(3).InfoS("Too many json patch operations", "count", len(patchObj))
			return nil, response.ErrTooManyJsonPatchOperations(maxJSONPatchOperations)
		}
		patchedJS, err := patchObj.Apply(versionedJS)
		if err != nil {
			klog.V(3).InfoS("Failed to apply json patch", "err", err)
			return nil, response.ErrMalformedJSON
		}
		return patchedJS, nil
	case types.MergePatchType:
		patchedJS, err = jsonpatch.MergePatch(versionedJS, patchBytes)
		if err != nil {
			klog.V(3).InfoS("Failed to apply json merge patch", "err", err)
			return nil, response.ErrMalformedJSON
		}
		return patchedJS, err
	default:
		// only here as a safety net - gin filters content-type
		return nil, fmt.Errorf("unknown Content-Type header for patch: %v", patchType)
	}
}
