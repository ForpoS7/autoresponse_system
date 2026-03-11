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
     * Сохранение полного состояния сессии (storage state) в JSON формате
     * @param userId ID пользователя
     * @param storageState JSON состояние сессии (cookies, localStorage, etc.)
     */
    @Transactional
    public void saveSessionState(Long userId, String storageState) {
        log.info("Сохранение состояния сессии для пользователя: {}", userId);

        if (storageState == null || storageState.isBlank()) {
            log.warn("Попытка сохранить пустое состояние сессии для пользователя: {}", userId);
            return;
        }

        Optional<HhToken> existing = hhTokenRepository.findByUserId(userId);

        if (existing.isPresent()) {
            HhToken hhToken = existing.get();
            hhToken.setTokenValue(storageState);
            hhTokenRepository.save(hhToken);
            log.info("Состояние сессии обновлено для пользователя: {}", userId);
        } else {
            HhToken hhToken = HhToken.builder()
                    .userId(userId)
                    .tokenValue(storageState)
                    .build();
            hhTokenRepository.save(hhToken);
            log.info("Состояние сессии сохранено для пользователя: {}", userId);
        }
    }

    /**
     * Получение токена для пользователя
     * @param userId ID пользователя
     * @return токен или empty если не найдена
     */
    public Optional<String> getToken(Long userId) {
        return hhTokenRepository.findByUserId(userId)
                .map(HhToken::getTokenValue);
    }

    /**
     * Получение состояния сессии (storage state) для пользователя
     * @param userId ID пользователя
     * @return JSON состояние сессии или empty если не найдена
     */
    public Optional<String> getSessionState(Long userId) {
        return hhTokenRepository.findByUserId(userId)
                .map(HhToken::getTokenValue);
    }

    /**
     * Проверка наличия токена у пользователя
     */
    public boolean hasToken(Long userId) {
        return hhTokenRepository.findByUserId(userId)
                .map(token -> token.getTokenValue() != null && !token.getTokenValue().isEmpty())
                .orElse(false);
    }
}
