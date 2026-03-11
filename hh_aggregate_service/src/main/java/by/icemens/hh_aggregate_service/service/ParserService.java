package by.icemens.hh_aggregate_service.service;

import by.icemens.hh_aggregate_service.config.PlaywrightConfig;
import by.icemens.hh_aggregate_service.dto.VacancyResponse;
import by.icemens.hh_aggregate_service.entity.Vacancy;
import by.icemens.hh_aggregate_service.message.VacancyMessage;
import by.icemens.hh_aggregate_service.publish.VacancyPublisher;
import by.icemens.hh_aggregate_service.repository.UserRepository;
import com.microsoft.playwright.BrowserContext;
import com.microsoft.playwright.ElementHandle;
import com.microsoft.playwright.Page;
import com.microsoft.playwright.options.WaitUntilState;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.security.core.userdetails.UserDetails;
import org.springframework.stereotype.Service;

import java.util.ArrayList;
import java.util.List;

@Service
@RequiredArgsConstructor
@Slf4j
public class ParserService {

    private final PlaywrightConfig playwrightConfig;
    private final PlaywrightService playwrightService;
    private final VacancyPublisher vacancyPublisher;
    private final UserRepository userRepository;
    private final TokenService tokenService;

    public List<Vacancy> parseVacancies(String query, int page, Long userId) {
        log.info("Парсинг вакансий: запрос='{}', страница={}, userId={}", query, page, userId);

        try (var browserPage = playwrightService.getPage(userId)) {
            Page pg = browserPage.get();

            String url = String.format(
                    "https://hh.ru/search/vacancy?text=%s&area=%s&items_on_page=20&page=%d",
                    query,
                    playwrightConfig.getAreaCode(),
                    page
            );

            log.info("URL: {}", url);

            // Переход на страницу с ожиданием DOMContentLoaded (быстрее чем full load)
            pg.navigate(url, new Page.NavigateOptions().setWaitUntil(WaitUntilState.DOMCONTENTLOADED));

            // Проверка на капчу
//            String pageTitle = pg.title();
//            if (pageTitle.toLowerCase().contains("captcha") ||
//                pg.content().toLowerCase().contains("robot")) {
//                throw new RuntimeException("Сработала защита от ботов (captcha). Обновите сессию.");
//            }

            // Ожидание появления вакансий
            pg.waitForSelector("[data-qa='vacancy-serp__vacancy']",
                    new Page.WaitForSelectorOptions().setTimeout(10000));

            List<Vacancy> vacancies = new ArrayList<>();
            List<ElementHandle> cards = pg.querySelectorAll("[data-qa='vacancy-serp__vacancy']");

            for (ElementHandle card : cards) {
                try {
                    ElementHandle titleEl = card.querySelector("[data-qa='serp-item__title']");
                    if (titleEl == null) continue;

                    String href = titleEl.getAttribute("href");
                    String title = titleEl.textContent().trim();

                    ElementHandle employerEl = card.querySelector(
                            "[data-qa='vacancy-serp__vacancy-employer']"
                    );
                    String employer = employerEl != null
                            ? employerEl.textContent().trim()
                            : "Не указан";

                    Vacancy vacancy = Vacancy.builder()
                            .title(title)
                            .url(href)
                            .employer(employer)
                            .userId(userId)
                            .build();

                    vacancies.add(vacancy);

                } catch (Exception e) {
                    log.warn("Не удалось распарсить вакансию: {}", e.getMessage());
                } finally {
                    if (card != null) card.dispose();
                }
            }

            log.info("Найдено вакансий: {}", vacancies.size());

            Vacancy vacanciesFirst = vacancies.getFirst();
            vacancyPublisher.publish(VacancyMessage.builder()
                    .vacancyId(vacanciesFirst.getId())
                    .title(vacanciesFirst.getTitle())
                    .employer(vacanciesFirst.getEmployer())
                    .url(vacanciesFirst.getUrl())
                    .parsedAt(vacanciesFirst.getCreatedAt())
                    .userId(vacanciesFirst.getUserId())
                    .build()
            );

            return vacancies;

        } catch (Exception e) {
            log.error("Ошибка при парсинге: {}", e.getMessage(), e);
            throw new RuntimeException("Ошибка при парсинге вакансий: " + e.getMessage(), e);
        }
    }

    public VacancyResponse toResponse(Vacancy vacancy) {
        return VacancyResponse.builder()
                .id(vacancy.getId())
                .title(vacancy.getTitle())
                .url(vacancy.getUrl())
                .employer(vacancy.getEmployer())
                .description(vacancy.getDescription())
                .salaryFrom(vacancy.getSalaryFrom())
                .salaryTo(vacancy.getSalaryTo())
                .currency(vacancy.getCurrency())
                .region(vacancy.getRegion())
                .build();
    }

    public Long getCurrentUserId(UserDetails userDetails) {
        return userRepository.findByEmail(userDetails.getUsername()).orElseThrow(
                () -> new IllegalStateException(
                        "Пользователь с таким email - " + userDetails.getUsername() + " не найден."
                )
        ).getId();
    }

    /**
     * Извлечение и сохранение hhtoken из cookies браузера
     * Открывает страницу входа hh.ru и ждёт авторизации пользователя
     * @param userId ID пользователя
     */
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
            tokenService.saveSessionState(userId, storageState);
            log.info("[OK] Состояние сессии сохранено в БД для пользователя: {}", userId);

        } catch (Exception e) {
            log.error("Ошибка при извлечении токена: {}", e.getMessage(), e);
            throw new RuntimeException("Ошибка при извлечении токена: " + e.getMessage(), e);
        }
    }
}
