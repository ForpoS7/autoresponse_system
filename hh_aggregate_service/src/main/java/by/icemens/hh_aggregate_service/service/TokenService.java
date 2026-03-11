package by.icemens.hh_aggregate_service.service;

import by.icemens.hh_aggregate_service.entity.HhToken;
import by.icemens.hh_aggregate_service.repository.HhTokenRepository;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.util.Optional;

@Service
@RequiredArgsConstructor
@Slf4j
public class TokenService {

    private final HhTokenRepository hhTokenRepository;

    /**
     * Сохранение сессии Playwright (JSON) для пользователя
     * @param userId ID пользователя
     * @param sessionJson JSON сессии от Playwright (storage state)
     */
    @Transactional
    public void saveHhSession(Long userId, String sessionJson) {
        log.info("Сохранение сессии HH.ru для пользователя: {}", userId);

        Optional<HhToken> existing = hhTokenRepository.findByUserId(userId);

        if (existing.isPresent()) {
            HhToken hhToken = existing.get();
            hhToken.setTokenValue(sessionJson);
            hhTokenRepository.save(hhToken);
            log.info("Сессия обновлена");
        } else {
            HhToken hhToken = HhToken.builder()
                    .userId(userId)
                    .tokenValue(sessionJson)
                    .build();
            hhTokenRepository.save(hhToken);
            log.info("Сессия сохранена");
        }
    }

    /**
     * Получение сессии Playwright (JSON) для пользователя
     * @param userId ID пользователя
     * @return JSON сессии или empty если не найдена
     */
    public Optional<String> getHhSession(Long userId) {
        return hhTokenRepository.findByUserId(userId)
                .map(HhToken::getTokenValue);
    }

    /**
     * Проверка наличия сессии у пользователя
     */
    public boolean hasHhSession(Long userId) {
        return hhTokenRepository.findByUserId(userId)
                .map(token -> token.getTokenValue() != null && !token.getTokenValue().isEmpty())
                .orElse(false);
    }
}
