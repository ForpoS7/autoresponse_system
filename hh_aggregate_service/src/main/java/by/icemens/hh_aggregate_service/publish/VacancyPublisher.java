package by.icemens.hh_aggregate_service.publish;

import by.icemens.hh_aggregate_service.dto.Vacancy;
import by.icemens.hh_aggregate_service.message.VacanciesBatch;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.kafka.core.KafkaTemplate;
import org.springframework.stereotype.Component;

import java.util.List;
import java.util.UUID;

@Component
@RequiredArgsConstructor
@Slf4j
public class VacancyPublisher {

    private final KafkaTemplate<String, Object> kafkaTemplate;
    private static final String TOPIC = "vacancies.parsed";

    /**
     * Публикует вакансии с jobId для корреляции.
     * Go-сервис ждёт именно тот jobId, который сам отправил в parse.requested.
     */
    public void publish(List<Vacancy> vacancies, String jobId) {
        Long userId = vacancies.isEmpty() ? null : vacancies.get(0).getUserId();
        VacanciesBatch batch = VacanciesBatch.builder()
                .jobId(jobId)
                .userId(userId)
                .vacancies(vacancies)
                .build();

        kafkaTemplate.send(TOPIC, jobId, batch)
                .whenComplete((result, ex) -> {
                    if (ex == null) {
                        log.info("vacancies.parsed published: jobId={}, count={}", jobId, vacancies.size());
                    } else {
                        log.error("Failed to publish vacancies.parsed jobId={}: {}", jobId, ex.getMessage());
                    }
                });
    }

    /** REST-эндпоинт вызывает без jobId — генерируем случайный. */
    public void publish(List<Vacancy> vacancies) {
        publish(vacancies, UUID.randomUUID().toString());
    }
}
