package by.icemens.hh_aggregate_service.publish;

import by.icemens.hh_aggregate_service.message.VacancyMessage;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.kafka.core.KafkaTemplate;
import org.springframework.stereotype.Component;

@Component
@RequiredArgsConstructor
@Slf4j
public class VacancyPublisher {

    private final KafkaTemplate<String, VacancyMessage> kafkaTemplate;
    private static final String TOPIC = "vacancies.parsed";

    public void publish(VacancyMessage vacancy) {
        kafkaTemplate.send(TOPIC, vacancy)
                .whenComplete((result, ex) -> {
                    if (ex == null) {
                        log.info("Вакансия отправлена в {}",
                                TOPIC);
                    } else {
                        log.error("Ошибка отправки вакансии : {}",
                                ex.getMessage());
                    }
                });
    }
}
