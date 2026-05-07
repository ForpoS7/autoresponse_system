package by.icemens.hh_aggregate_service.kafka;

import by.icemens.hh_aggregate_service.message.ParseRequestMessage;
import by.icemens.hh_aggregate_service.service.ParserService;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.kafka.annotation.KafkaListener;
import org.springframework.stereotype.Component;

@Component
@RequiredArgsConstructor
@Slf4j
public class ParseRequestConsumer {

    private final ParserService parserService;

    /**
     * Слушает parse.requested — Go-сервис просит распарсить вакансии.
     * Результат публикуется в vacancies.parsed с тем же jobId для корреляции.
     */
    @KafkaListener(
            topics = "parse.requested",
            groupId = "aggregate-service",
            containerFactory = "parseRequestListenerFactory"
    )
    public void consume(ParseRequestMessage message) {
        log.info("parse.requested received: jobId={}, query='{}', userId={}",
                message.getJobId(), message.getQuery(), message.getUserId());
        try {
            parserService.parseVacancies(
                    message.getQuery(),
                    message.getPage(),
                    message.getUserId(),
                    message.getJobId()
            );
        } catch (Exception e) {
            log.error("Failed to process parse request jobId={}: {}", message.getJobId(), e.getMessage());
        }
    }
}
