package by.icemens.hh_aggregate_service.message;

import by.icemens.hh_aggregate_service.dto.Vacancy;
import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

import java.util.List;

/**
 * Kafka-сообщение для топика vacancies.parsed.
 * jobId связывает запрос на парсинг (parse.requested) с результатом,
 * что позволяет Go-сервису ждать именно своих вакансий.
 */
@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class VacanciesBatch {
    private String jobId;
    private Long userId;
    private List<Vacancy> vacancies;
}
