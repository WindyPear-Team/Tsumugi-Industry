package maintenance

import (
	"time"
	"tsumugi-industry/internal/models"
)

func (s *Service) CleanupAuditLogs(days int) (int64, error) {
	if days < 1 {
		days = 30
	}
	cutoff := time.Now().AddDate(0, 0, -days)
	result := s.db.Where("created_at < ?", cutoff).Delete(&models.AuditLog{})
	return result.RowsAffected, result.Error
}
