package by.icemens.hh_aggregate_service.service;

import by.icemens.hh_aggregate_service.config.PlaywrightConfig;
import by.icemens.hh_aggregate_service.dto.Vacancy;
import by.icemens.hh_aggregate_service.publish.VacancyPublisher;
import com.microsoft.playwright.ElementHandle;
import com.microsoft.playwright.Page;
import com.microsoft.playwright.options.WaitUntilState;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
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

                    Long vacancyId = Long.valueOf(href.replaceFirst(".*/vacancy/", "").split("\\?")[0]);

                    Vacancy vacancy = Vacancy.builder()
                            .id(vacancyId)
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

            vacancyPublisher.publish(vacancies);

            return vacancies;

        } catch (Exception e) {
            log.error("Ошибка при парсинге: {}", e.getMessage(), e);
            throw new RuntimeException("Ошибка при парсинге вакансий: " + e.getMessage(), e);
        }
    }

}
