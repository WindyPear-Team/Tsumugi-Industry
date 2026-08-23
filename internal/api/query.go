package api

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Page struct {
	Page      int   `json:"page"`
	PageSize  int   `json:"page_size"`
	Total     int64 `json:"total"`
	PageCount int   `json:"page_count"`
}

func paginate(query *gorm.DB, c *gin.Context) (*gorm.DB, Page) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	if size > 100 {
		size = 100
	}
	var total int64
	query.Count(&total)
	count := int(total) / size
	if total%int64(size) != 0 {
		count++
	}
	return query.Offset((page - 1) * size).Limit(size), Page{Page: page, PageSize: size, Total: total, PageCount: count}
}

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
