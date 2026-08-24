package api

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
	"tsumugi-industry/internal/models"
)

type flowFunctionRequest struct {
	Code        string `json:"code"`
	Name        string `json:"name" binding:"required,max=128"`
	Description string `json:"description"`
	ReturnType  string `json:"return_type"`
	Parameters  any    `json:"parameters"`
	Definition  any    `json:"definition"`
}

func (r *Router) flowFunctions(c *gin.Context) {
	var items []models.FlowFunction
	if err := r.db.Order("id DESC").Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (r *Router) flowFunction(c *gin.Context) {
	var item models.FlowFunction
	if err := r.db.First(&item, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "flow function not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"function": item})
}

func encodeJSON(value any) (string, error) {
	if value == nil {
		return "[]", nil
	}
	encoded, err := json.Marshal(value)
	return string(encoded), err
}

func (r *Router) createFlowFunction(c *gin.Context) {
	var req flowFunctionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	parameters, err := encodeJSON(req.Parameters)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	definition, err := encodeJSON(req.Definition)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	code := strings.TrimSpace(req.Code)
	if code == "" {
		code = "FN_" + strings.ToUpper(strings.NewReplacer(" ", "_", "-", "_").Replace(strings.TrimSpace(req.Name)))
	}
	returnType := strings.ToLower(strings.TrimSpace(req.ReturnType))
	if returnType == "" {
		returnType = "none"
	}
	item := models.FlowFunction{Code: code, Name: strings.TrimSpace(req.Name), Description: req.Description, ReturnType: returnType, Parameters: parameters, Definition: definition}
	if err := r.db.Create(&item).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "函数编码已存在或无法创建"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"function": item})
}

func (r *Router) updateFlowFunction(c *gin.Context) {
	var req flowFunctionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var item models.FlowFunction
	if err := r.db.First(&item, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "flow function not found"})
		return
	}
	parameters, err := encodeJSON(req.Parameters)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	definition, err := encodeJSON(req.Definition)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	code := strings.TrimSpace(req.Code)
	if code == "" {
		code = item.Code
	}
	returnType := strings.ToLower(strings.TrimSpace(req.ReturnType))
	if returnType == "" {
		returnType = "none"
	}
	if err := r.db.Model(&item).Updates(map[string]any{"code": code, "name": strings.TrimSpace(req.Name), "description": req.Description, "return_type": returnType, "parameters": parameters, "definition": definition}).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	r.db.First(&item, item.ID)
	c.JSON(http.StatusOK, gin.H{"function": item})
}

func (r *Router) deleteFlowFunction(c *gin.Context) {
	if err := r.db.Delete(&models.FlowFunction{}, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
