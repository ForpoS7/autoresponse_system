package by.icemens.hh_aggregate_service.service;

import by.icemens.hh_aggregate_service.dto.TokenRequest;
import by.icemens.hh_aggregate_service.entity.HhToken;
import by.icemens.hh_aggregate_service.repository.HhTokenRepository;
import com.microsoft.playwright.BrowserContext;
import com.microsoft.playwright.Page;
import com.microsoft.playwright.options.WaitUntilState;
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
    private final PlaywrightService playwrightService;

    /**
     * Сохранение полного состояния сессии (storage state) в JSON формате
     * @param userId ID пользователя
     * @param storageState JSON состояние сессии (cookies, localStorage, etc.)
     */
    @Transactional
    protected void saveSessionState(Long userId, String storageState) {
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
     * Извлечение и сохранение hhtoken из cookies браузера
     * Открывает страницу входа hh.ru и ждёт авторизации пользователя
     * @param userId ID пользователя
     */
    @Transactional
    public void extractAndSaveToken(Long userId) {
        log.info("Извлечение hhtoken для пользователя: {}", userId);

        try (var browserPage = playwrightService.getPage(userId)) {
            Page pg = browserPage.get();
            BrowserContext context = browserPage.getContext();

            // Переход на страницу входа
            log.info("Переход на страницу входа hh.ru...");
            pg.navigate("https://hh.ru/login", new Page.NavigateOptions().setWaitUntil(WaitUntilState.DOMCONTENTLOADED));

            // Ожидание пока пользователь не авторизуется (максимум 5 минут)
            log.info("Ожидание авторизации пользователя (максимум 5 минут)...");
            log.info("Откройте браузер и войдите в аккаунт hh.ru");

            // Ждём появления элемента профиля (признак авторизации)
            try {
                pg.waitForSelector("[data-qa='header-profile']", new Page.WaitForSelectorOptions().setTimeout(300000));
                log.info("Пользователь авторизован!");
            } catch (Exception e) {
                log.warn("Таймаут ожидания авторизации. Проверяем наличие hhtoken...");
            }

            // Сохранение полного состояния сессии (storage state)
            log.info("Сохранение состояния сессии...");
            String storageState = context.storageState();
            saveSessionState(userId, storageState);
            log.info("[OK] Состояние сессии сохранено в БД для пользователя: {}", userId);

        } catch (Exception e) {
            log.error("Ошибка при извлечении токена: {}", e.getMessage(), e);
            throw new RuntimeException("Ошибка при извлечении токена: " + e.getMessage(), e);
        }
    }

    /**
     * Получение токена для пользователя
     * @param userId ID пользователя
     * @return токен или empty если не найдена
     */
    public TokenRequest getToken(Long userId, String email) {
        return hhTokenRepository.findByUserId(userId)
                .map(this::toTokenRequest).orElseThrow(
                        () -> new IllegalStateException(
                                "У пользователя с таким email - " + email + " не нашлось токена."
                ));
    }

    public TokenRequest toTokenRequest(HhToken token) {
        return TokenRequest.builder()
                .userId(token.getUserId())
                .tokenValue(token.getTokenValue())
                .build();
    }
}
