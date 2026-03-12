package by.icemens.hh_aggregate_service.publish;

import by.icemens.hh_aggregate_service.dto.Vacancy;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.kafka.core.KafkaTemplate;
import org.springframework.stereotype.Component;

import java.util.List;

@Component
@RequiredArgsConstructor
@Slf4j
public class VacancyPublisher {

    private final KafkaTemplate<Object, List<Vacancy>> kafkaTemplate;
    private static final String TOPIC = "vacancies.parsed";

    public void publish(List<Vacancy> vacancies) {
        kafkaTemplate.send(TOPIC, vacancies)
                .whenComplete((result, ex) -> {
                    if (ex == null) {
                        log.info("Вакансии отправлены в {}",
                                TOPIC);
                    } else {
                        log.error("Ошибка отправки вакансий : {}",
                                ex.getMessage());
                    }
                });
    }
}
