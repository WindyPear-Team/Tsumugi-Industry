package api

import (
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// applyTableQuery keeps filtering and ordering server-side while only allowing
// columns explicitly declared by each endpoint.
func applyTableQuery(query *gorm.DB, c *gin.Context, sortable, filters map[string]string, defaultSort string) *gorm.DB {
	for parameter, column := range filters {
		if value := strings.TrimSpace(c.Query(parameter)); value != "" {
			if strings.HasSuffix(parameter, "_is_active") || strings.HasSuffix(parameter, "_status_code") {
				query = query.Where(column+" = ?", value)
				continue
			}
			query = query.Where(column+" LIKE ?", "%"+value+"%")
		}
	}

	column, ok := sortable[c.DefaultQuery("sort_by", defaultSort)]
	if !ok {
		column = sortable[defaultSort]
	}
	direction := "ASC"
	if strings.EqualFold(c.Query("sort_order"), "desc") {
		direction = "DESC"
	}
	return query.Order(column + " " + direction)
}
