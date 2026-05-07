package by.icemens.hh_aggregate_service.publish;

import by.icemens.hh_aggregate_service.message.TokenUpdatedMessage;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.kafka.core.KafkaTemplate;
import org.springframework.stereotype.Component;

@Component
@RequiredArgsConstructor
@Slf4j
public class TokenPublisher {

    private final KafkaTemplate<String, Object> kafkaTemplate;
    private static final String TOPIC = "token.updated";

    /**
     * Публикует событие обновления токена. Go-сервис слушает этот топик
     * и обновляет локальный кеш — больше не нужен HTTP-вызов к Java.
     */
    public void publish(Long userId, String tokenValue) {
        TokenUpdatedMessage msg = TokenUpdatedMessage.builder()
                .userId(userId)
                .tokenValue(tokenValue)
                .build();

        kafkaTemplate.send(TOPIC, String.valueOf(userId), msg)
                .whenComplete((result, ex) -> {
                    if (ex == null) {
                        log.info("token.updated published for userId={}", userId);
                    } else {
                        log.error("Failed to publish token.updated for userId={}: {}", userId, ex.getMessage());
                    }
                });
    }
}
