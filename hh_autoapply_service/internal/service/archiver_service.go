package service

import (
	"context"
	"encoding/json"
	"fmt"
	"hh_autoapply_service/internal/repository"
	"hh_autoapply_service/pkg/storage"
	"log"
	"time"
)

// ArchiverService реализует трёхуровневое хранение данных:
//
//	Hot  — PostgreSQL, данные до logsRetainDays / vacsRetainDays
//	Warm — PostgreSQL (те же таблицы, запросы по дате фильтруют)
//	Cold — MinIO (S3), данные старше порога → JSON-батчи
type ArchiverService struct {
	autoApplyRepo  *repository.AutoApplyRepository
	vacancyRepo    *repository.VacancyRepository
	storage        *storage.MinIOClient
	logsRetainDays int
	vacsRetainDays int
}

type ArchiveResult struct {
	ArchivedLogs      int    `json:"archived_logs"`
	ArchivedVacancies int    `json:"archived_vacancies"`
	StorageKey        string `json:"storage_key,omitempty"`
	Error             string `json:"error,omitempty"`
}

func NewArchiverService(
	autoApplyRepo *repository.AutoApplyRepository,
	vacancyRepo *repository.VacancyRepository,
	storage *storage.MinIOClient,
	logsRetainDays, vacsRetainDays int,
) *ArchiverService {
	return &ArchiverService{
		autoApplyRepo:  autoApplyRepo,
		vacancyRepo:    vacancyRepo,
		storage:        storage,
		logsRetainDays: logsRetainDays,
		vacsRetainDays: vacsRetainDays,
	}
}

// ArchiveLogs переносит auto_apply_logs старше logsRetainDays в MinIO.
// Hot → Cold: SELECT old → upload JSON → DELETE from PostgreSQL
func (s *ArchiverService) ArchiveLogs(ctx context.Context) (int, error) {
	logs, err := s.autoApplyRepo.GetOldLogs(s.logsRetainDays)
	if err != nil {
		return 0, fmt.Errorf("get old logs: %w", err)
	}
	if len(logs) == 0 {
		log.Printf("[archiver] no logs older than %d days", s.logsRetainDays)
		return 0, nil
	}

	data, err := json.Marshal(logs)
	if err != nil {
		return 0, fmt.Errorf("marshal logs: %w", err)
	}

	key := coldKey("logs", len(logs))
	if err := s.storage.Upload(ctx, key, data); err != nil {
		return 0, fmt.Errorf("upload logs: %w", err)
	}

	ids := make([]int64, len(logs))
	for i, l := range logs {
		ids[i] = l.ID
	}
	if err := s.autoApplyRepo.DeleteLogsByIDs(ids); err != nil {
		// Данные уже в MinIO — логируем но не падаем, повторная архивация создаст дубль
		log.Printf("[archiver] WARN: logs uploaded but delete failed: %v", err)
	}

	log.Printf("[archiver] archived %d logs → %s", len(logs), key)
	return len(logs), nil
}

// ArchiveVacancies переносит vacancies старше vacsRetainDays в MinIO.
func (s *ArchiverService) ArchiveVacancies(ctx context.Context) (int, error) {
	vacancies, err := s.vacancyRepo.GetOldVacancies(s.vacsRetainDays)
	if err != nil {
		return 0, fmt.Errorf("get old vacancies: %w", err)
	}
	if len(vacancies) == 0 {
		log.Printf("[archiver] no vacancies older than %d days", s.vacsRetainDays)
		return 0, nil
	}

	data, err := json.Marshal(vacancies)
	if err != nil {
		return 0, fmt.Errorf("marshal vacancies: %w", err)
	}

	key := coldKey("vacancies", len(vacancies))
	if err := s.storage.Upload(ctx, key, data); err != nil {
		return 0, fmt.Errorf("upload vacancies: %w", err)
	}

	ids := make([]string, len(vacancies))
	for i, v := range vacancies {
		ids[i] = v.ID
	}
	if err := s.vacancyRepo.DeleteVacanciesByIDs(ids); err != nil {
		log.Printf("[archiver] WARN: vacancies uploaded but delete failed: %v", err)
	}

	log.Printf("[archiver] archived %d vacancies → %s", len(vacancies), key)
	return len(vacancies), nil
}

// RunAll запускает полный цикл архивации.
func (s *ArchiverService) RunAll(ctx context.Context) ArchiveResult {
	logs, err := s.ArchiveLogs(ctx)
	if err != nil {
		return ArchiveResult{Error: err.Error()}
	}
	vacs, err := s.ArchiveVacancies(ctx)
	if err != nil {
		return ArchiveResult{ArchivedLogs: logs, Error: err.Error()}
	}
	return ArchiveResult{ArchivedLogs: logs, ArchivedVacancies: vacs}
}

// ListCold возвращает список файлов в cold storage по префиксу.
func (s *ArchiverService) ListCold(ctx context.Context, prefix string) ([]storage.ObjectInfo, error) {
	return s.storage.List(ctx, "cold/"+prefix)
}

// StartScheduler запускает архивацию по расписанию (каждые intervalHours часов).
func (s *ArchiverService) StartScheduler(ctx context.Context, intervalHours int) {
	interval := time.Duration(intervalHours) * time.Hour
	go func() {
		log.Printf("[archiver] scheduler started, interval=%dh", intervalHours)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				log.Println("[archiver] scheduled run started")
				result := s.RunAll(ctx)
				log.Printf("[archiver] scheduled run done: logs=%d vacancies=%d err=%s",
					result.ArchivedLogs, result.ArchivedVacancies, result.Error)
			}
		}
	}()
}

// coldKey формирует путь: cold/{type}/YYYY/MM/DD/batch_{count}_{ts}.json
func coldKey(dataType string, count int) string {
	now := time.Now().UTC()
	return fmt.Sprintf("cold/%s/%d/%02d/%02d/batch_%d_%d.json",
		dataType, now.Year(), now.Month(), now.Day(), count, now.UnixMilli())
}
